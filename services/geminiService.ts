
import { GoogleGenAI, Modality } from "@google/genai";
import { logToServer } from "../utils/logger";
import prompts from "../prompts.json";

// Enhanced cleaner to remove all metadata and whitespace
const cleanBase64 = (base64: string) => {
  return base64.replace(/^data:(image|audio|video)\/[\w-+\.]+;base64,/, '').replace(/\s/g, '');
};

const getAI = () => {
  const apiKey = process.env.API_KEY;
  if (!apiKey) {
    throw new Error("请先选择并连接 API Key。");
  }
  return new GoogleGenAI({ apiKey });
};

// Helper to construct prompt from JSON
const buildPrompt = (type: 'composite' | 'edit' | 'upscale', lang: 'en' | 'zh' = 'en', variables?: Record<string, string>) => {
  const p = prompts[type];
  let promptText = "";

  if (type === 'composite') {
    const c = p as typeof prompts.composite;
    promptText = `
      ROLE: ${c.role[lang]}
      TASK: ${c.task[lang]}

      ANATOMY & IDENTITY RULES (CRITICAL):
      ${c.rules.anatomy[lang]}

      COMPOSITION & LIGHTING:
      ${c.rules.composition[lang]}
      
      OUTPUT: ${c.output[lang]}
    `;
  } else if (type === 'edit') {
    const e = p as typeof prompts.edit;
    let task = e.task[lang];
    if (variables?.userPrompt) {
        task = task.replace("{userPrompt}", variables.userPrompt);
    }
    promptText = `
      ROLE: ${e.role[lang]}
      TASK: ${task}
      
      STRICT CONSTRAINTS:
      ${e.constraints[lang]}
    `;
  } else if (type === 'upscale') {
    const u = p as typeof prompts.upscale;
    promptText = `
      ROLE: ${u.role[lang]}
      TASK: ${u.task[lang]}
      
      INSTRUCTIONS:
      ${u.instructions[lang]}
    `;
  }

  return promptText.trim();
};

export const generateComposite = async (bgBase64: string, selfieBase64: string): Promise<string> => {
  const ai = getAI();
  const startTime = Date.now();
  
  try {
    // Use English prompts by default for better adherence with Gemini models, 
    // but structure supports switching to 'zh' if needed.
    const prompt = buildPrompt('composite', 'en');

    logToServer("Google API Request - Generate Composite", { 
      model: 'gemini-3-pro-image-preview',
      prompt
    });

    const response = await ai.models.generateContent({
      model: 'gemini-3-pro-image-preview',
      contents: {
        parts: [
          { text: prompt },
          { 
            inlineData: { 
              mimeType: 'image/jpeg', 
              data: cleanBase64(bgBase64) 
            } 
          },
          { 
            inlineData: { 
              mimeType: 'image/jpeg', 
              data: cleanBase64(selfieBase64) 
            } 
          },
        ],
      },
      config: {
        temperature: 0.2,
        topP: 0.3,
        seed: 42,
        responseModalities: [Modality.IMAGE],
        imageConfig: {
           aspectRatio: "16:9", 
           imageSize: "2K",     
        }
      },
    });

    logToServer("Google API Response - Generate Composite", {
      durationMs: Date.now() - startTime,
      candidatesCount: response.candidates?.length,
      usageMetadata: response.usageMetadata
    });

    const parts = response.candidates?.[0]?.content?.parts;
    if (parts) {
      for (const part of parts) {
        if (part.inlineData) {
          return `data:image/png;base64,${part.inlineData.data}`;
        }
      }
    }
    throw new Error("Gemini 未能生成有效的图像内容。");
  } catch (error: any) {
    logToServer("Google API Error - Generate Composite", { message: error.message, stack: error.stack }, "ERROR");
    console.error("Gemini Composition Error:", error);
    if (error.message === "Failed to fetch") {
      throw new Error("Gemini 服务连接失败，请检查 API Key 或网络环境。");
    }
    throw error;
  }
};

export const editImage = async (imageBase64: string, userPrompt: string): Promise<string> => {
  const ai = getAI();
  const startTime = Date.now();

  try {
    const prompt = buildPrompt('edit', 'en', { userPrompt });

    logToServer("Google API Request - Edit Image", { 
      model: 'gemini-3-pro-image-preview',
      userPrompt,
      systemPrompt: prompt
    });

    const response = await ai.models.generateContent({
      model: 'gemini-3-pro-image-preview',
      contents: {
        parts: [
          { text: prompt },
          { 
            inlineData: { 
              mimeType: 'image/png', 
              data: cleanBase64(imageBase64) 
            } 
          },
        ],
      },
      config: {
        temperature: 0.2,
        topP: 0.3,
        seed: 42,
        responseModalities: [Modality.IMAGE],
        imageConfig: {
           aspectRatio: "16:9", 
           imageSize: "2K",     
        }
      },
    });

    logToServer("Google API Response - Edit Image", {
      durationMs: Date.now() - startTime,
      candidatesCount: response.candidates?.length,
      usageMetadata: response.usageMetadata
    });

    const parts = response.candidates?.[0]?.content?.parts;
    if (parts) {
      for (const part of parts) {
        if (part.inlineData) {
          return `data:image/png;base64,${part.inlineData.data}`;
        }
      }
    }
    throw new Error("重绘处理未能生成新图像。");
  } catch (error: any) {
    logToServer("Google API Error - Edit Image", { message: error.message, stack: error.stack }, "ERROR");
    console.error("Gemini Edit Error:", error);
    throw error;
  }
};

export const upscaleImage = async (imageBase64: string): Promise<string> => {
  const ai = getAI();
  const startTime = Date.now();

  try {
    const prompt = buildPrompt('upscale', 'en');

    logToServer("Google API Request - Upscale Image", { 
      model: 'gemini-3-pro-image-preview',
      prompt
    });

    const response = await ai.models.generateContent({
      model: 'gemini-3-pro-image-preview',
      contents: {
        parts: [
          { text: prompt },
          { 
            inlineData: { 
              mimeType: 'image/png', 
              data: cleanBase64(imageBase64) 
            } 
          },
        ],
      },
      config: {
        temperature: 0.2,
        topP: 0.3,
        seed: 42,
        responseModalities: [Modality.IMAGE],
        imageConfig: {
            imageSize: '2K', 
            aspectRatio: "16:9" 
        }
      },
    });

    logToServer("Google API Response - Upscale Image", {
      durationMs: Date.now() - startTime,
      candidatesCount: response.candidates?.length,
      usageMetadata: response.usageMetadata
    });

    const parts = response.candidates?.[0]?.content?.parts;
    if (parts) {
      for (const part of parts) {
        if (part.inlineData) {
          return `data:image/png;base64,${part.inlineData.data}`;
        }
      }
    }
    throw new Error("超分处理未能成功。");
  } catch (error: any) {
    logToServer("Google API Error - Upscale Image", { message: error.message, stack: error.stack }, "ERROR");
    console.error("Gemini Upscale Error:", error);
    throw error;
  }
};

export const transcribeAudio = async (audioBase64: string): Promise<string> => {
  const ai = getAI();
  const startTime = Date.now();

  try {
    // For transcription, usually English instruction works best even for multilingual audio
    const prompt = prompts.transcribe.prompt.en; 

    logToServer("Google API Request - Transcribe Audio", { 
      model: 'gemini-2.5-flash',
      prompt
    });

    const response = await ai.models.generateContent({
      model: 'gemini-2.5-flash',
      contents: {
        parts: [
          { text: prompt },
          {
            inlineData: {
              mimeType: 'audio/webm',
              data: cleanBase64(audioBase64)
            }
          }
        ]
      },
      config: {
        temperature: 0,
        topP: 0.3,
      }
    });

    logToServer("Google API Response - Transcribe Audio", {
      durationMs: Date.now() - startTime,
      text: response.text?.trim()
    });

    return response.text?.trim() || "";
  } catch (error: any) {
    logToServer("Google API Error - Transcribe Audio", { message: error.message, stack: error.stack }, "ERROR");
    console.error("Gemini Transcription Error:", error);
    throw new Error("语音识别失败，请检查麦克风权限或重试。");
  }
};
