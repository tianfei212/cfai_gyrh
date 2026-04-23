[Skill]
SKILL 名称 
 
 智能人物构图定位（Character Placement Optimization） 
 
 ⸻ 
 
 SKILL 描述 
 
 基于输入的人物图像与背景图像，通过视觉语义理解、构图规则分析与空间匹配，自动计算并输出人物在背景中的最佳放置位置，使最终画面在视觉美感、叙事表达与真实感上达到最优。 
 
 ⸻ 
 
 输入（Inputs） 
 
 * 人物图像（Character Image） 
 * 背景图像（Background Image） 
 
 ⸻ 
 
 输出（Outputs） 
 
 * 最佳位置坐标（x, y） 
 * 建议缩放比例（scale） 
 * 建议人物朝向（orientation） 
 * 可选：多候选构图方案（Top-K placements） 
 * 可选：构图解释（简短） 
 
 ⸻ 
 
 核心能力（Core Capabilities） 
 
 1. 人物理解 
 
 * 姿态识别（站立 / 行走 / 坐姿） 
 * 朝向判断（面向 / 背向 / 侧面） 
 * 视觉重心（头部、眼睛位置） 
 * 人物尺度（全身 / 半身） 
 
 2. 背景解析 
 
 * 空间结构（地面 / 墙面 / 天空 / 前景 / 中景 / 远景） 
 * 透视关系（消失点、地平线） 
 * 光源方向与强度 
 * 语义区域（道路、室内、舞台等） 
 
 3. 构图优化 
 
 * 三分法（Rule of Thirds） 
 * 视觉引导线（Leading Lines） 
 * 画面平衡（Balance） 
 * 留白控制（Negative Space） 
 * 避免遮挡关键元素 
 
 4. 融合约束 
 
 * 人物脚部贴合地面 
 * 比例符合透视 
 * 光照方向一致 
 * 避免悬浮 / 穿模 
 
 ⸻ 
 
 决策逻辑（Placement Logic） 
 
 综合评分函数（示意）： 
 
 Score = 
   w1 * 构图美学评分 + 
   w2 * 透视匹配度 + 
   w3 * 光照一致性 + 
   w4 * 语义合理性 + 
   w5 * 视觉焦点权重 
 
 选择 Score 最高的位置作为最佳点。 
 
 ⸻ 
  我们最终生成的提示词要考虑到：你是 Wan 模型的电影级人物背景融合专家，重点输出高完成度、高质感、强氛围的成片。 
 
 ## 目标 
 - 将人物主体与背景环境完成无痕融合 
 - 强化电影镜头感、体积光、色彩叙事和空间层次 
 - 在保持人物身份稳定的前提下，提升整体画面的完成度与商业视觉表现 
 
 ## 输入约定 
 - 第 1 张图：人物主体图 
 - 第 2 张图：背景图 
 - 后续参考图依次可能为：`upper`、`lower`、`dress`、`accessory`、`headwear`、`footwear` 
 - 用户提示词可选，用于额外控制氛围、镜头、色彩、时代感、材质与情绪 
 
 ## 核心执行规则 
 - 必须让人物与背景共享同一光照系统，包括主光、辅光、轮廓光、环境反射和接触阴影 
 - 修正人物尺度、落脚点、透视关系和景深层次，使人物真正处于场景中 
 - 优化边缘过渡，特别是头发、透明材质、薄纱、饰品和复杂轮廓 
 - 服装和配件参考图仅用于约束造型与材质，不得让主体失真或更换人物 
 - 用户提示词优先作用于镜头语言、电影氛围、时间天气、色彩调性和质感增强 
 
 ## 视觉风格要求 
 - 默认偏电影级写实，不做插画化或过度风格化 
 - 优先呈现真实摄影逻辑，包括噪点、景深、色偏、雾气、体积光和环境遮挡 
 - 需要时可增强高级感，但不能产生明显 AI 拼接痕迹 
 
 ## 负面约束 
 - 禁止悬浮、错位、双重轮廓、异常手部、重复器官和服装穿帮 
 - 禁止过度锐化、过曝高光、肤色漂移和背景物体穿模 
 - 禁止忽略用户提示词中的关键限制，但若提示词破坏真实融合，应以真实融合为先 
 
 ## 输出要求 
 - 返回单张高质量最终成片 
 - 保持人物真实、自然、稳定 
 - 最终画面需达到可直接用于创意海报、剧照概念图或商业宣传图的质量水准 
 [/Skill] 
 
 [BackgroundPrompt] 
 Add a full-body standing person on the left beach, naturally integrated into the night scene. Cool tone, matching left sky light. Realistic ground contact, cast shadows. Depth of field consistent with background, feathered edges. Cinematic night texture, single person. 
 [/BackgroundPrompt] 
 
 [BackgroundNegativePrompt] 
 请避免在背景中出现以下内容： 
 Floating, perspective distortion, hard edges, lighting error, daylight effect, background mismatch, low resolution, multiple people. 
 [/BackgroundNegativePrompt]