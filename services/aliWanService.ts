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
 * 阿里 Wan 2.6 姿态生成服务 (同步模式)
 * 
 * 功能: 姿态迁移 / 人物重绘
 * 输入:
 *   - personImageBase64: 源人物图 (提供身份和服装)
 *   - poseImageBase64: 姿态参考图 (提供动作)
 *   - bgImageBase64: 背景图 (提供场景)
 *   - userPrompt: 用户提示词
 */
export const generateWanPose = async (
  personImageBase64: string,
  poseImageBase64: string,
  bgImageBase64: string,
  userPrompt?: string
): Promise<string> => {
  if (!ALIBABA_API_KEY) {
    throw new Error("未配置 ALI_API_KEY，请在 .env.local 中配置。");
  }

  try {
    const submitUrl = `${BASE_URL}/services/aigc/multimodal-generation/generation`;
    
    // 构建姿态生成专用 Prompt
    const poseConfig = prompts.aliWanPose;
    const constraints = Object.values(poseConfig.constraints)
      .map((c: any) => c.zh)
      .join("。");
      
    const finalPrompt = `${userPrompt || poseConfig.default_prompt}。${constraints}`;

    logToServer("[DEBUG] AliWan Pose Prompt:", finalPrompt);
    
    const requestBody = {
      model: "wan2.6-image",
      input: {
        messages: [
          {
            role: "user",
            content: [
              { text: finalPrompt },
              { image: `data:image/png;base64,${cleanBase64(personImageBase64)}` }, // 图1：源人物
              { image: `data:image/png;base64,${cleanBase64(poseImageBase64)}` },   // 图2：姿态参考
              { image: `data:image/png;base64,${cleanBase64(bgImageBase64)}` }      // 图3：背景场景
            ]
          }
        ]
      },
      parameters: {
        prompt_extend: true,
        watermark: false,
        n: 1,
        enable_interleave: false,
        size: "1024*1024"
      }
    };

    logToServer("[DEBUG] AliWan Pose Request URL:", submitUrl);
    
    const headers = {
        'Authorization': `Bearer ${ALIBABA_API_KEY}`,
        'Content-Type': 'application/json'
    };

    const submitResponse = await fetch(`${PROXY_URL}${encodeURIComponent(submitUrl)}`, {
      method: 'POST',
      headers: headers,
      body: JSON.stringify(requestBody)
    });

    logToServer(`[DEBUG] AliWan Pose Response Status: ${submitResponse.status} ${submitResponse.statusText}`);
    
    const textResponse = await submitResponse.text();
    logToServer(`[DEBUG] AliWan Pose Raw Response Body (First 500 chars): ${textResponse.substring(0, 500)}...`);

    let submitData;
    try {
      submitData = JSON.parse(textResponse);
    } catch (e) {
      logToServer("AliWan Pose Response Parsing Error. Raw Response:", textResponse, "ERROR");
      throw new Error(`AliWan 响应解析失败: ${textResponse.substring(0, 200)}...`);
    }

    if (!submitResponse.ok) {
      logToServer("AliWan Pose API Rejection Details:", JSON.stringify(submitData, null, 2), "ERROR");
      const errorCode = submitData.code || "";
      const errorMsg = submitData.message || "请求被阿里云拒绝";
      throw new Error(`阿里 Wan 2.6 接口异常 (${submitResponse.status}): ${errorCode} - ${errorMsg}`);
    }

    if (submitData.output?.choices?.[0]?.message?.content) {
        const content = submitData.output.choices[0].message.content;
        const resultItem = content.find((c: any) => c.image);
        const resultUrl = resultItem?.image;

        if (!resultUrl) {
            throw new Error("任务成功但未找到图像 URL");
        }
        
        logToServer("[DEBUG] AliWan Pose Result URL Found:", resultUrl);
        
        // Use Local Proxy
        const localProxyUrl = `/api/proxy-image?url=${encodeURIComponent(resultUrl)}`;
        let downloadResponse = await fetch(localProxyUrl);
        
        if (downloadResponse.status === 404) {
             const publicProxyUrl = `${PROXY_URL}${encodeURIComponent(resultUrl)}`;
             downloadResponse = await fetch(publicProxyUrl);
        }

        if (!downloadResponse.ok) {
            const errorText = await downloadResponse.text();
            throw new Error(`结果图像下载失败: ${downloadResponse.status} ${downloadResponse.statusText}`);
        }
        
        const blob = await downloadResponse.blob();
        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onloadend = () => resolve(reader.result as string);
            reader.onerror = () => reject(new Error("Base64 转换失败"));
            reader.readAsDataURL(blob);
        });
    }

    throw new Error("未收到图像结果");
  } catch (error: any) {
    console.error("AliWan Pose Service Detailed Error:", error);
    logToServer("AliWan Pose Service Error:", error.message, "ERROR");
    throw error;
  }
};
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
        console.log("Received Task ID, entering polling mode (Unexpected based on curl example)");
    } else {
        throw new Error("未收到图像结果也未收到任务 ID");
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

/**
 * 阿里 Wan 2.6 图像编辑/风格迁移服务
 * 
 * 功能: 基于提示词编辑图像
 * 输入:
 *   - imageBase64: 原图
 *   - userPrompt: 编辑/风格提示词
 */
export const editWanImage = async (
  imageBase64: string,
  userPrompt: string
): Promise<string> => {
  if (!ALIBABA_API_KEY) {
    throw new Error("未配置 ALI_API_KEY，请在 .env.local 中配置。");
  }

  try {
    const submitUrl = `${BASE_URL}/services/aigc/multimodal-generation/generation`;
    
    logToServer("[DEBUG] AliWan Edit Prompt:", userPrompt);
    
    const requestBody = {
      model: "wan2.6-image",
      input: {
        messages: [
          {
            role: "user",
            content: [
              { text: userPrompt },
              { image: `data:image/png;base64,${cleanBase64(imageBase64)}` }
            ]
          }
        ]
      },
      parameters: {
        prompt_extend: true,
        watermark: false,
        n: 1,
        enable_interleave: false,
        size: "1024*1024"
      }
    };

    logToServer("[DEBUG] AliWan Edit Request URL:", submitUrl);

    const headers = {
        'Authorization': `Bearer ${ALIBABA_API_KEY}`,
        'Content-Type': 'application/json'
    };

    const submitResponse = await fetch(`${PROXY_URL}${encodeURIComponent(submitUrl)}`, {
      method: 'POST',
      headers: headers,
      body: JSON.stringify(requestBody)
    });

    logToServer(`[DEBUG] AliWan Edit Response Status: ${submitResponse.status} ${submitResponse.statusText}`);
    
    const textResponse = await submitResponse.text();
    logToServer(`[DEBUG] AliWan Edit Raw Response Body (First 500 chars): ${textResponse.substring(0, 500)}...`);

    let submitData;
    try {
      submitData = JSON.parse(textResponse);
    } catch (e) {
      logToServer("AliWan Edit Response Parsing Error. Raw Response:", textResponse, "ERROR");
      throw new Error(`AliWan 响应解析失败: ${textResponse.substring(0, 200)}...`);
    }

    if (!submitResponse.ok) {
      logToServer("AliWan Edit API Rejection Details:", JSON.stringify(submitData, null, 2), "ERROR");
      const errorCode = submitData.code || "";
      const errorMsg = submitData.message || "请求被阿里云拒绝";
      throw new Error(`阿里 Wan 2.6 接口异常 (${submitResponse.status}): ${errorCode} - ${errorMsg}`);
    }

    if (submitData.output?.choices?.[0]?.message?.content) {
        const content = submitData.output.choices[0].message.content;
        const resultItem = content.find((c: any) => c.image);
        const resultUrl = resultItem?.image;

        if (!resultUrl) {
            throw new Error("任务成功但未找到图像 URL");
        }
        
        logToServer("[DEBUG] AliWan Edit Result URL Found:", resultUrl);
        
        // Download logic (Reuse local proxy strategy)
        let blob: Blob;
        try {
            logToServer("[DEBUG] Attempting Direct Download:", resultUrl);
            const directResponse = await fetch(resultUrl);
            if (directResponse.ok) {
                blob = await directResponse.blob();
            } else {
                throw new Error(`Direct download failed: ${directResponse.status}`);
            }
        } catch (directError) {
            logToServer("Direct download failed, falling back to proxy...", directError);
            const localProxyUrl = `/api/proxy-image?url=${encodeURIComponent(resultUrl)}`;
            let downloadResponse = await fetch(localProxyUrl);
            if (downloadResponse.status === 404) {
                 const publicProxyUrl = `${PROXY_URL}${encodeURIComponent(resultUrl)}`;
                 downloadResponse = await fetch(publicProxyUrl);
            }
            if (!downloadResponse.ok) {
                throw new Error(`结果图像下载失败: ${downloadResponse.status}`);
            }
            blob = await downloadResponse.blob();
        }

        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onloadend = () => resolve(reader.result as string);
            reader.onerror = () => reject(new Error("Base64 转换失败"));
            reader.readAsDataURL(blob);
        });
    }

    throw new Error("未收到图像结果");
  } catch (error: any) {
    console.error("AliWan Edit Service Detailed Error:", error);
    logToServer("AliWan Edit Service Error:", error.message, "ERROR");
    throw error;
  }
};
