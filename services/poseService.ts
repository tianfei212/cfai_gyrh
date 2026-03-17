import { generateWanPose as aliWanPose, generateWanImage as aliWanComposite } from './aliWanService';
import { generateGeminiPose } from './geminiService';

export type PoseServiceType = 'aliWan' | 'google';

/**
 * 通用姿态生成服务接口
 * 
 * @param personImage 源人物图像 (Base64) - 提供身份/服装
 * @param poseImage 姿态参考图像 (Base64) - 提供动作骨架
 * @param bgImage 背景图像 (Base64) - 提供场景环境
 * @param serviceType 服务提供商 ('aliWan' | 'google')
 * @param prompt 可选提示词
 * @returns 生成的图像 Base64
 */
export const generatePose = async (
  personImage: string,
  poseImage: string,
  bgImage: string,
  serviceType: PoseServiceType = 'aliWan',
  prompt?: string
): Promise<string> => {
  console.log(`[PoseService] Generating pose using ${serviceType}...`);
  
  if (serviceType === 'aliWan') {
    return await aliWanPose(personImage, poseImage, bgImage, prompt);
  } else if (serviceType === 'google') {
    return await generateGeminiPose(personImage, poseImage, bgImage, prompt);
  }
  
  throw new Error(`未知的服务类型: ${serviceType}`);
};

/**
 * 通用图像合成服务接口 (用于保持接口一致性)
 */
export const generateComposite = async (
  personImage: string,
  bgImage: string,
  serviceType: PoseServiceType = 'aliWan',
  prompt?: string
): Promise<string> => {
    if (serviceType === 'aliWan') {
        return await aliWanComposite(personImage, bgImage, prompt);
    }
    // Google implementation is direct call in App.tsx currently, can be unified here later
    throw new Error("Google 合成服务请直接调用 geminiService");
}
