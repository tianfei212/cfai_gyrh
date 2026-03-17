// @ts-ignore
const ALIBABA_API_KEY = process.env.ALI_API_KEY || "";
const BASE_URL = "https://dashscope.aliyuncs.com/api/v1";

// 使用代理以绕过浏览器 CORS 限制
const PROXY_URL = "https://corsproxy.io/?";

const cleanBase64 = (base64: string) => {
  if (!base64) return "";
  return base64.replace(/^data:image\/[\w-+\.]+;base64,/, '').replace(/[\n\r\s]/g, '');
};

const delay = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));

/**
 * 阿里 Wan 2.6 图像生成服务 (异步模式)
 * 模型标识符: wan2.6-image
 * 
 * 功能: 图像编辑 / 图像合成
 * 输入: 
 *   - personImageBase64: 从摄像头抠出的人像 (前景)
 *   - backgroundImageBase64: 背景图
 *   - prompt: 提示词 (默认为"将人物自然融入背景中")
 */
export const generateWanImage = async (
  personImageBase64: string, 
  backgroundImageBase64: string, 
  prompt: string = "将人物自然融入背景中"
): Promise<string> => {
  if (!ALIBABA_API_KEY) {
    throw new Error("未配置 ALI_API_KEY，请在 .env.local 中配置。");
  }

  try {
    const submitUrl = `${BASE_URL}/services/aigc/multimodal-generation/generation`;
    
    // 1. 提交异步处理任务
    // 注意: 根据用户要求，第一张图为摄像头人像，第二张图为背景
    const requestBody = {
      model: "wan2.6-image",
      input: {
        messages: [
          {
            role: "user",
            content: [
              { text: prompt },
              { image: `data:image/png;base64,${cleanBase64(personImageBase64)}` },
              { image: `data:image/png;base64,${cleanBase64(backgroundImageBase64)}` }
            ]
          }
        ]
      },
      parameters: {
        enable_interleave: false, // 图像编辑模式
        size: "1K", // 默认 1K 分辨率
        n: 1 // 生成一张图
      }
    };

    console.log("[DEBUG] AliWan Request URL:", submitUrl);
    console.log("[DEBUG] AliWan Request Headers:", {
      'Authorization': `Bearer ${ALIBABA_API_KEY.substring(0, 6)}...`,
      'Content-Type': 'application/json',
      'X-DashScope-Async': 'enable'
    });
    // Truncate Base64 in logs for readability
    const debugBody = JSON.parse(JSON.stringify(requestBody));
    if(debugBody.input?.messages?.[0]?.content) {
       debugBody.input.messages[0].content.forEach((item: any) => {
         if(item.image) item.image = "[BASE64_IMAGE_DATA_TRUNCATED]";
       });
    }
    console.log("[DEBUG] AliWan Request Body:", JSON.stringify(debugBody, null, 2));

    const submitResponse = await fetch(`${PROXY_URL}${encodeURIComponent(submitUrl)}`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${ALIBABA_API_KEY}`,
        'Content-Type': 'application/json',
        'X-DashScope-Async': 'enable' // 必须开启异步模式
      },
      body: JSON.stringify(requestBody)
    });

    console.log("[DEBUG] AliWan Response Status:", submitResponse.status, submitResponse.statusText);

    // Handle non-JSON responses from proxy or server
    const textResponse = await submitResponse.text();
    console.log("[DEBUG] AliWan Raw Response:", textResponse);

    let submitData;
    try {
      submitData = JSON.parse(textResponse);
    } catch (e) {
      console.error("AliWan Response Parsing Error. Raw Response:", textResponse); // Print FULL response
      throw new Error(`AliWan 响应解析失败 (可能是代理错误或鉴权失败): ${textResponse.substring(0, 200)}...`);
    }

    if (!submitResponse.ok) {
      console.error("AliWan API Rejection Details:", JSON.stringify(submitData, null, 2));
      const errorCode = submitData.code || "";
      const errorMsg = submitData.message || "请求被阿里云拒绝";
      throw new Error(`阿里 Wan 2.6 接口异常 (${submitResponse.status}): ${errorCode} - ${errorMsg}`);
    }

    const taskId = submitData.output?.task_id;
    if (!taskId) {
      throw new Error(`任务提交失败: ${submitData.message || "未获取到有效的 Task ID"}`);
    }

    // 2. 轮询处理结果
    let attempts = 0;
    const maxAttempts = 60; // 轮询 60 次，每次间隔 2 秒，约 2 分钟超时

    while (attempts < maxAttempts) {
      const statusUrl = `${BASE_URL}/tasks/${taskId}`;
      const statusResponse = await fetch(`${PROXY_URL}${encodeURIComponent(statusUrl)}`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${ALIBABA_API_KEY}`
        }
      });

      if (statusResponse.ok) {
        const statusData = await statusResponse.json();
        const taskStatus = statusData.output?.task_status;

        if (taskStatus === 'SUCCEEDED') {
          // 获取生成结果
          // 注意: 响应结构中 choices[0].message.content[0].image 是结果 URL
          const choices = statusData.output?.choices;
          if (!choices || choices.length === 0) {
             throw new Error("任务成功但未返回结果图像");
          }
          
          const resultUrl = choices[0].message?.content?.find((c: any) => c.image)?.image;
          
          if (!resultUrl) {
             throw new Error("任务成功但未找到图像 URL");
          }

          // 3. 通过代理下载结果图像以避免 CORS
          const downloadResponse = await fetch(`${PROXY_URL}${encodeURIComponent(resultUrl)}`);
          if (!downloadResponse.ok) throw new Error("结果图像下载失败，请检查网络连接。");
          
          const blob = await downloadResponse.blob();
          return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onloadend = () => resolve(reader.result as string);
            reader.onerror = () => reject(new Error("Base64 转换失败"));
            reader.readAsDataURL(blob);
          });

        } else if (taskStatus === 'FAILED') {
          throw new Error(`阿里 Wan 2.6 生成失败: ${statusData.output?.message || "后端处理异常"}`);
        } else if (taskStatus === 'CANCELED') {
           throw new Error("任务被取消");
        }
        // PENDING or RUNNING, continue polling
      }

      await delay(2000); // 间隔 2 秒
      attempts++;
    }

    throw new Error("任务处理超时，请重试或查看控制台任务历史记录。");

  } catch (error: any) {
    console.error("AliWan Service Detailed Error:", error);
    
    if (error.message === "Failed to fetch") {
      throw new Error("由于 CORS 限制或网络波动，请求未能成功发出。请确保已在阿里后台开通「Wan 2.6」模型权限。");
    }
    
    throw error;
  }
};
