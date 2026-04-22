export const screens = [
  { key: 'dashboard', label: '主工作台' },
  { key: 'history', label: '历史记录' },
  { key: 'backgrounds', label: '背景管理' },
  { key: 'preview', label: '全屏预览', hideInNav: true },
  { key: 'capture', label: '摄像头拍摄', hideInNav: true },
  { key: 'rendering', label: '生成中', hideInNav: true },
  { key: 'login', label: '登录', hideInNav: true },
  { key: 'logout', label: '退出确认', hideInNav: true },
];

export const galleryCards = [
  { name: '新春灯影', tone: 'blue' },
  { name: '未来通道', tone: 'red' },
  { name: '星港春帆', tone: 'blue' },
  { name: '云海古亭', tone: 'red' },
  { name: '金色露台', tone: 'blue' },
  { name: '晨雾海岸', tone: 'blue' },
];

export const historyCards = Array.from({ length: 12 }, (_, index) => ({
  id: String(index + 1).padStart(3, '0'),
  tone: index % 2 === 0 ? 'blue' : 'red',
}));

export const backgroundRows = [
  {
    id: '001',
    tone: 'blue',
    wan: 'cinematic night alley, wet pavement reflections, soft rim light, ultra detailed',
    gemini: 'moori cyberpunk street, volumetric fog, high contrast, photorealistic',
  },
  {
    id: '002',
    tone: 'red',
    wan: 'ancient temple in clouds, golden sunrise rays, detailed stone texture',
    gemini: 'floating sanctuary above clouds, warm sunlight beams, highly detailed',
  },
  {
    id: '003',
    tone: 'blue',
    wan: 'misty riverside winter bridge, frozen river, soft blue ambience, 4k',
    gemini: 'snow bridge at dusk, cinematic blue zone, atmospheric perspective',
  },
  {
    id: '004',
    tone: 'red',
    wan: 'desert canyon at golden hour, long shadows, aerial composition',
    gemini: 'golden canyon vista, warm cinematic lighting, crisp detail',
  },
  {
    id: '005',
    tone: 'blue',
    wan: 'futuristic harbor skyline, neon reflections, volumetric haze',
    gemini: 'sci fi harbor city at night, reflective water, high fidelity',
  },
];

export const styleTags = [
  '电影感',
  '写实',
  '油画',
  '柔焦暖光',
  '迷雾',
  '黑金色调',
  '逆光边缘',
  '像素',
  '国风',
  '写实',
  '轻复古',
  '冷光检色',
  '赛博城',
  '雾性地',
  '青铜绿',
  '相机',
];
