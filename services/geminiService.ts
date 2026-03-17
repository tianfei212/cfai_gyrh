
import { logToServer } from "../utils/logger";
import prompts from "../prompts.json";

// Enhanced cleaner to remove all metadata and whitespace
const cleanBase64 = (base64: string) => {
  return base64.replace(/^data:(image|audio|video)\/[\w-+\.]+;base64,/, '').replace(/\s/g, '');
};

const postJson = async (url: string, body: any) => {
  const r = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  });
  if (!r.ok) {
    const err = await r.json().catch(() => ({}));
    throw new Error(err?.error || '请求失败');
  }
  return r.json();
};

// Helper to construct prompt from JSON
const buildPrompt = (type: 'composite' | 'edit' | 'upscale' | 'geminiPose', lang: 'en' | 'zh' = 'en', variables?: Record<string, string>) => {
  const p = prompts[type];
  let promptText = "";

  if (type === 'geminiPose') {
    const g = p as typeof prompts.geminiPose;
    promptText = `
      ROLE: ${g.role[lang]}
      TASK: ${g.task[lang]}
      
      INSTRUCTIONS:
      ${g.instructions[lang]}
    `;
  } else if (type === 'composite') {
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

export const generateGeminiPose = async (
  personImageBase64: string,
  poseImageBase64: string,
  bgImageBase64: string,
  userPrompt?: string
): Promise<string> => {
  const startTime = Date.now();
  
  try {
    const prompt = buildPrompt('geminiPose' as any, 'en', { userPrompt: userPrompt || '' });

    logToServer("Google API Request - Generate Pose", { 
      model: 'gemini-3-pro-image-preview',
      prompt
    });

    // Reuse /api/compose endpoint but with different logic if needed
    // Or create a new endpoint /api/pose if backend supports it
    // For now, assuming backend /api/compose can handle 2 images + prompt
    const data = await postJson('/api/compose', { 
        bgBase64: bgImageBase64, // Use actual background
        selfieBase64: personImageBase64,
        poseBase64: poseImageBase64, // Pass pose image as additional param
        prompt // Send specific pose prompt
    });
    
    logToServer("Google API Response - Generate Pose", { durationMs: Date.now() - startTime });
    if (data?.base64) return data.base64;
    throw new Error("未获取到生成结果。");
  } catch (error: any) {
    logToServer("Google API Error - Generate Pose", { message: error.message, stack: error.stack }, "ERROR");
    console.error("Gemini Pose Error:", error);
    throw error;
  }
};

export const generateComposite = async (bgBase64: string, selfieBase64: string): Promise<string> => {
  const startTime = Date.now();
  
  try {
    const prompt = buildPrompt('composite', 'en');

    logToServer("Google API Request - Generate Composite", { 
      model: 'gemini-3-pro-image-preview',
      prompt
    });

    const data = await postJson('/api/compose', { bgBase64, selfieBase64 });
    logToServer("Google API Response - Generate Composite", { durationMs: Date.now() - startTime });
    if (data?.base64) return data.base64;
    throw new Error("未获取到生成结果。");
  } catch (error: any) {
    logToServer("Google API Error - Generate Composite", { message: error.message, stack: error.stack }, "ERROR");
    console.error("Gemini Composition Error:", error);
    throw error;
  }
};

export const editImage = async (imageBase64: string, userPrompt: string): Promise<string> => {
  const startTime = Date.now();

  try {
    const prompt = buildPrompt('edit', 'en', { userPrompt });

    logToServer("Google API Request - Edit Image", { 
      model: 'gemini-3-pro-image-preview',
      userPrompt,
      systemPrompt: prompt
    });

    const data = await postJson('/api/edit', { imageBase64, prompt: userPrompt });
    logToServer("Google API Response - Edit Image", { durationMs: Date.now() - startTime });
    if (data?.base64) return data.base64;
    throw new Error("未获取到编辑结果。");
  } catch (error: any) {
    logToServer("Google API Error - Edit Image", { message: error.message, stack: error.stack }, "ERROR");
    console.error("Gemini Edit Error:", error);
    throw error;
  }
};

export const upscaleImage = async (imageBase64: string): Promise<string> => {
  const startTime = Date.now();

  try {
    const prompt = buildPrompt('upscale', 'en');

    logToServer("Google API Request - Upscale Image", { 
      model: 'gemini-3-pro-image-preview',
      prompt
    });

    const data = await postJson('/api/upscale', { imageBase64 });
    logToServer("Google API Response - Upscale Image", { durationMs: Date.now() - startTime });
    if (data?.base64) return data.base64;
    throw new Error("未获取到超分结果。");
  } catch (error: any) {
    logToServer("Google API Error - Upscale Image", { message: error.message, stack: error.stack }, "ERROR");
    console.error("Gemini Upscale Error:", error);
    throw error;
  }
};

export const transcribeAudio = async (audioBase64: string): Promise<string> => {
  const startTime = Date.now();

  try {
    const prompt = prompts.transcribe.prompt.en; 

    logToServer("Google API Request - Transcribe Audio", { 
      model: 'gemini-2.5-flash',
      prompt
    });

    const data = await postJson('/api/transcribe', { audioBase64 });
    logToServer("Google API Response - Transcribe Audio", { durationMs: Date.now() - startTime });
    return data?.text || "";
  } catch (error: any) {
    logToServer("Google API Error - Transcribe Audio", { message: error.message, stack: error.stack }, "ERROR");
    console.error("Gemini Transcription Error:", error);
    throw new Error("语音识别失败，请检查麦克风权限或重试。");
  }
};
