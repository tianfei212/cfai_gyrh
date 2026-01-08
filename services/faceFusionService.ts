
const ALIBABA_API_KEY = "sk-f529fce2b0d44c54b8704bb96383a040";
const BASE_URL = "https://dashscope.aliyuncs.com/api/v1";

// 使用代理以绕过浏览器 CORS 限制
const PROXY_URL = "https://corsproxy.io/?";

const cleanBase64 = (base64: string) => {
  if (!base64) return "";
  return base64.replace(/^data:image\/[\w-+\.]+;base64,/, '').replace(/[\n\r\s]/g, '');
};

const delay = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));

/**
 * 阿里 DashScope 人脸融合服务
 * 模型标识符: face-fusion-v1
 * 
 * 重要提示：如果您遇到 "Model not exist" 错误，是因为账号未激活此模型。
 * 请执行以下步骤：
 * 1. 登录阿里云 DashScope (灵积) 控制台。
 * 2. 进入「模型库」，搜索并找到「人脸融合」。
 * 3. 点击「开通服务」或「开通模型」。
 * 4. 确保您的 API Key (sk-...) 处于正常状态。
 */
export const faceFusion = async (templateBase64: string, sourceBase64: string): Promise<string> => {
  try {
    const submitUrl = `${BASE_URL}/services/aigc/face-fusion/face-fusion`;
    
    // 1. 提交异步处理任务
    const submitResponse = await fetch(`${PROXY_URL}${encodeURIComponent(submitUrl)}`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${ALIBABA_API_KEY}`,
        'Content-Type': 'application/json',
        'X-DashScope-Async': 'enable'
      },
      body: JSON.stringify({
        model: "face-fusion-v1", 
        input: {
          template_image: cleanBase64(templateBase64),
          source_image: cleanBase64(sourceBase64)
        }
      })
    });

    const submitData = await submitResponse.json();

    if (!submitResponse.ok) {
      console.error("Alibaba API Rejection Details:", JSON.stringify(submitData, null, 2));
      
      const errorCode = submitData.code || "";
      const errorMsg = submitData.message || "请求被阿里云拒绝";
      
      // 捕获模型未激活的核心错误
      if (errorMsg.toLowerCase().includes("model not exist") || errorCode === "InvalidParameter") {
        throw new Error("【阿里模型未就绪】\n1. 请确认已在阿里云灵积(DashScope)控制台开通「人脸融合」服务。\n2. 请确认所使用的 API Key (sk-...) 正确无误。\n注意：此错误通常是因为未手动开通对应模型。");
      }
      
      throw new Error(`阿里接口异常 (${submitResponse.status}): ${errorMsg}`);
    }

    const taskId = submitData.output?.task_id;
    if (!taskId) {
      throw new Error(`任务提交失败: ${submitData.message || "未获取到有效的 Task ID"}`);
    }

    // 2. 轮询处理结果
    let attempts = 0;
    const maxAttempts = 60; 

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
          const resultUrl = statusData.output.output_image_url;
          
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
          throw new Error(`阿里融合失败: ${statusData.output?.message || "后端处理异常"}`);
        }
      }

      await delay(1500); 
      attempts++;
    }

    throw new Error("任务处理超时，请重试或查看控制台任务历史记录。");
  } catch (error: any) {
    console.error("Face Fusion Detailed Error:", error);
    
    if (error.message === "Failed to fetch") {
      throw new Error("由于 CORS 限制或网络波动，请求未能成功发出。请确保已在阿里后台开通「人脸融合」模型权限。");
    }
    
    throw error;
  }
};
