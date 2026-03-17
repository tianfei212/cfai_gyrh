import { logToServer } from '../utils/logger';

import prompts from '../prompts.json';

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
 * 阿里 Wan 2.6 图像生成服务 (同步模式)
 * 模型标识符: wan2.6-image
 * 
 * 功能: 图像编辑 / 图像合成
 * 输入: 
 *   - personImageBase64: 从摄像头抠出的人像 (前景)
 *   - backgroundImageBase64: 背景图
 *   - userPrompt: 用户可选的提示词，如果为空将使用 prompts.json 中的严格预设
 */
export const generateWanImage = async (
  personImageBase64: string, 
  backgroundImageBase64: string, 
  userPrompt?: string
): Promise<string> => {
  if (!ALIBABA_API_KEY) {
    throw new Error("未配置 ALI_API_KEY，请在 .env.local 中配置。");
  }

  try {
    const submitUrl = `${BASE_URL}/services/aigc/multimodal-generation/generation`;
    
    // 构建严格的控制提示词
    const aliWanConfig = prompts.aliWan;
    const constraints = Object.values(aliWanConfig.constraints)
      .map((c: any) => c.zh) // 优先使用中文约束
      .join("。");
      
    // 组合最终 Prompt：用户提示词 (如果有) + 默认任务描述 + 严格约束
    const finalPrompt = `${userPrompt || aliWanConfig.default_prompt}。${constraints}`;

    logToServer("[DEBUG] AliWan Final Prompt:", finalPrompt);
    
    // 1. 提交异步处理任务
    // 参考成功 curl 示例调整请求结构
    // 关键点：
    // 1. text 字段需明确 key 为 "text"
    // 2. image 字段需明确 key 为 "image"
    // 3. content 数组顺序: [text, image1, image2]
    // 4. parameters 参数调整: prompt_extend: true, watermark: false
    // 5. 移除 X-DashScope-Async 头部，改为同步调用（根据 curl 示例返回直接结果）
    
    const requestBody = {
      model: "wan2.6-image",
      input: {
        messages: [
          {
            role: "user",
            content: [
              { text: finalPrompt },
              { image: `data:image/png;base64,${cleanBase64(personImageBase64)}` },
              { image: `data:image/png;base64,${cleanBase64(backgroundImageBase64)}` }
            ]
          }
        ]
      },
      parameters: {
        prompt_extend: true,
        watermark: false,
        n: 1,
        enable_interleave: false,
        size: "1024*1024" // 修正 size 格式，文档通常为 1024*1024 或 1280*720
      }
    };

    logToServer("[DEBUG] AliWan Request URL:", submitUrl);
    // Remove Async Header for synchronous attempt based on curl example
    const headers = {
        'Authorization': `Bearer ${ALIBABA_API_KEY}`,
        'Content-Type': 'application/json'
    };
    
    logToServer("[DEBUG] AliWan Request Headers:", {
      ...headers,
      'Authorization': `Bearer ${ALIBABA_API_KEY.substring(0, 6)}...`
    });

    const submitResponse = await fetch(`${PROXY_URL}${encodeURIComponent(submitUrl)}`, {
      method: 'POST',
      headers: headers,
      body: JSON.stringify(requestBody)
    });

    logToServer(`[DEBUG] AliWan Response Status: ${submitResponse.status} ${submitResponse.statusText}`);
    
    const textResponse = await submitResponse.text();
    logToServer(`[DEBUG] AliWan Raw Response Body (First 500 chars): ${textResponse.substring(0, 500)}...`);

    let submitData;
    try {
      submitData = JSON.parse(textResponse);
    } catch (e) {
      logToServer("AliWan Response Parsing Error. Raw Response:", textResponse, "ERROR");
      throw new Error(`AliWan 响应解析失败: ${textResponse.substring(0, 200)}...`);
    }

    if (!submitResponse.ok) {
      logToServer("AliWan API Rejection Details:", JSON.stringify(submitData, null, 2), "ERROR");
      const errorCode = submitData.code || "";
      const errorMsg = submitData.message || "请求被阿里云拒绝";
      throw new Error(`阿里 Wan 2.6 接口异常 (${submitResponse.status}): ${errorCode} - ${errorMsg}`);
    }

    // Handle Synchronous Response (Direct Output)
    if (submitData.output?.choices?.[0]?.message?.content) {
        const content = submitData.output.choices[0].message.content;
        const resultItem = content.find((c: any) => c.image);
        const resultUrl = resultItem?.image;

        if (!resultUrl) {
            throw new Error("任务成功但未找到图像 URL");
        }
        
        logToServer("[DEBUG] AliWan Result URL Found:", resultUrl);
        
        // 阿里 OSS 返回的链接通常带有鉴权参数 (Signature, Expires 等)
        // 使用 corsproxy.io 代理时，如果直接对这种带参 URL 进行编码转发，可能会导致签名失效或被拒绝 (403)
        // 尝试:
        // 1. 直接下载 (如果浏览器支持 CORS)
        // 2. 如果直接下载失败，尝试使用 fetch mode: 'no-cors' (但这无法获取 blob)
        // 3. 关键修复: 对于 corsproxy.io，我们不需要对整个 URL 进行 component 编码，或者尝试不使用代理直接请求
        
        let blob: Blob;
        try {
            // 尝试直接请求 (最稳妥，如果 OSS 配置了 CORS)
            logToServer("[DEBUG] Attempting Direct Download:", resultUrl);
            const directResponse = await fetch(resultUrl);
            if (directResponse.ok) {
                blob = await directResponse.blob();
                logToServer("Direct download successful");
            } else {
                throw new Error(`Direct download failed: ${directResponse.status}`);
            }
        } catch (directError) {
            logToServer("Direct download failed, falling back to proxy...", directError);
            
            // 降级到代理
            // 优先使用本地 Vite 开发服务器提供的代理 (/api/proxy-image)
            // 只有在生产环境且无本地代理时才考虑 corsproxy.io (但 corsproxy.io 有 403 限制)
            
            // 本地代理地址
            const localProxyUrl = `/api/proxy-image?url=${encodeURIComponent(resultUrl)}`;
            logToServer("[DEBUG] Attempting Local Proxy Download:", localProxyUrl);

            let downloadResponse = await fetch(localProxyUrl);
            
            // 如果本地代理失败（例如生产环境没有此路由），尝试公共代理作为最后的救命稻草
            if (downloadResponse.status === 404) {
                 logToServer("[WARN] Local proxy not found, falling back to corsproxy.io");
                 const publicProxyUrl = `${PROXY_URL}${encodeURIComponent(resultUrl)}`;
                 downloadResponse = await fetch(publicProxyUrl);
            }
            
            // Log Proxy Headers for Debugging
            const headerObj: any = {};
            downloadResponse.headers.forEach((val, key) => { headerObj[key] = val; });
            logToServer("[DEBUG] Proxy Response Headers:", headerObj);

            if (!downloadResponse.ok) {
                const errorText = await downloadResponse.text();
                logToServer("[DEBUG] Proxy Download Error Body:", errorText, "ERROR");
                throw new Error(`结果图像下载失败 (Proxy Error): ${downloadResponse.status} ${downloadResponse.statusText}`);
            }
            
            blob = await downloadResponse.blob();
        }
        
        logToServer(`[DEBUG] Download Blob Size: ${blob.size}, Type: ${blob.type}`);

        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onloadend = () => resolve(reader.result as string);
            reader.onerror = () => reject(new Error("Base64 转换失败"));
            reader.readAsDataURL(blob);
        });
    }

    // Fallback to Async Task ID handling if response contains output.task_id (just in case)
    const taskId = submitData.output?.task_id;
    if (taskId) {
        // ... Existing polling logic ...
        // Re-implement polling logic here if needed, or assume sync for now as per curl
        console.log("Received Task ID, entering polling mode (Unexpected based on curl example)");
        // Reuse the polling logic below...
    } else {
        throw new Error("未收到图像结果也未收到任务 ID");
    }
    
    // 2. 轮询处理结果 (Only if we got a task_id, which means we need to restructure the code flow slightly)
    // To keep it simple, I will wrap the polling logic in a function or just put it after the check.
    
    // ... (Keep existing polling logic but make it conditional on taskId existence)


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
