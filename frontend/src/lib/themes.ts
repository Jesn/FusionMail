/**
 * 主题配置文件
 * 定义各主题的 CSS 变量值
 * 兼容 tweakcn.com 主题格式
 */

export interface ThemeColors {
  background: string;
  foreground: string;
  card: string;
  'card-foreground': string;
  popover: string;
  'popover-foreground': string;
  primary: string;
  'primary-foreground': string;
  secondary: string;
  'secondary-foreground': string;
  muted: string;
  'muted-foreground': string;
  accent: string;
  'accent-foreground': string;
  destructive: string;
  'destructive-foreground'?: string;
  border: string;
  input: string;
  ring: string;
  'chart-1'?: string;
  'chart-2'?: string;
  'chart-3'?: string;
  'chart-4'?: string;
  'chart-5'?: string;
  sidebar?: string;
  'sidebar-foreground'?: string;
  'sidebar-primary'?: string;
  'sidebar-primary-foreground'?: string;
  'sidebar-accent'?: string;
  'sidebar-accent-foreground'?: string;
  'sidebar-border'?: string;
  'sidebar-ring'?: string;
}

export interface Theme {
  name: string;
  label: string;
  light: ThemeColors;
  dark: ThemeColors;
}

// 默认主题（Neutral）
export const defaultTheme: Theme = {
  name: 'default',
  label: '默认',
  light: {
    background: 'oklch(1 0 0)',
    foreground: 'oklch(0.145 0 0)',
    card: 'oklch(1 0 0)',
    'card-foreground': 'oklch(0.145 0 0)',
    popover: 'oklch(1 0 0)',
    'popover-foreground': 'oklch(0.145 0 0)',
    primary: 'oklch(0.205 0 0)',
    'primary-foreground': 'oklch(0.985 0 0)',
    secondary: 'oklch(0.97 0 0)',
    'secondary-foreground': 'oklch(0.205 0 0)',
    muted: 'oklch(0.97 0 0)',
    'muted-foreground': 'oklch(0.556 0 0)',
    accent: 'oklch(0.97 0 0)',
    'accent-foreground': 'oklch(0.205 0 0)',
    destructive: 'oklch(0.577 0.245 27.325)',
    border: 'oklch(0.922 0 0)',
    input: 'oklch(0.922 0 0)',
    ring: 'oklch(0.708 0 0)',
    'chart-1': 'oklch(0.646 0.222 41.116)',
    'chart-2': 'oklch(0.6 0.118 184.704)',
    'chart-3': 'oklch(0.398 0.07 227.392)',
    'chart-4': 'oklch(0.828 0.189 84.429)',
    'chart-5': 'oklch(0.769 0.188 70.08)',
    sidebar: 'oklch(0.985 0 0)',
    'sidebar-foreground': 'oklch(0.145 0 0)',
    'sidebar-primary': 'oklch(0.205 0 0)',
    'sidebar-primary-foreground': 'oklch(0.985 0 0)',
    'sidebar-accent': 'oklch(0.97 0 0)',
    'sidebar-accent-foreground': 'oklch(0.205 0 0)',
    'sidebar-border': 'oklch(0.922 0 0)',
    'sidebar-ring': 'oklch(0.708 0 0)',
  },
  dark: {
    background: 'oklch(0.145 0 0)',
    foreground: 'oklch(0.985 0 0)',
    card: 'oklch(0.205 0 0)',
    'card-foreground': 'oklch(0.985 0 0)',
    popover: 'oklch(0.205 0 0)',
    'popover-foreground': 'oklch(0.985 0 0)',
    primary: 'oklch(0.922 0 0)',
    'primary-foreground': 'oklch(0.205 0 0)',
    secondary: 'oklch(0.269 0 0)',
    'secondary-foreground': 'oklch(0.985 0 0)',
    muted: 'oklch(0.269 0 0)',
    'muted-foreground': 'oklch(0.708 0 0)',
    accent: 'oklch(0.269 0 0)',
    'accent-foreground': 'oklch(0.985 0 0)',
    destructive: 'oklch(0.704 0.191 22.216)',
    border: 'oklch(1 0 0 / 10%)',
    input: 'oklch(1 0 0 / 15%)',
    ring: 'oklch(0.556 0 0)',
    'chart-1': 'oklch(0.488 0.243 264.376)',
    'chart-2': 'oklch(0.696 0.17 162.48)',
    'chart-3': 'oklch(0.769 0.188 70.08)',
    'chart-4': 'oklch(0.627 0.265 303.9)',
    'chart-5': 'oklch(0.645 0.246 16.439)',
    sidebar: 'oklch(0.205 0 0)',
    'sidebar-foreground': 'oklch(0.985 0 0)',
    'sidebar-primary': 'oklch(0.488 0.243 264.376)',
    'sidebar-primary-foreground': 'oklch(0.985 0 0)',
    'sidebar-accent': 'oklch(0.269 0 0)',
    'sidebar-accent-foreground': 'oklch(0.985 0 0)',
    'sidebar-border': 'oklch(1 0 0 / 10%)',
    'sidebar-ring': 'oklch(0.556 0 0)',
  },
};


// Tangerine 主题（橙色调）
export const tangerineTheme: Theme = {
  name: 'tangerine',
  label: '橘子',
  light: {
    background: 'oklch(0.9383 0.0042 236.4993)',
    foreground: 'oklch(0.3211 0 0)',
    card: 'oklch(1.0000 0 0)',
    'card-foreground': 'oklch(0.3211 0 0)',
    popover: 'oklch(1.0000 0 0)',
    'popover-foreground': 'oklch(0.3211 0 0)',
    primary: 'oklch(0.6397 0.1720 36.4421)',
    'primary-foreground': 'oklch(1.0000 0 0)',
    secondary: 'oklch(0.9670 0.0029 264.5419)',
    'secondary-foreground': 'oklch(0.4461 0.0263 256.8018)',
    muted: 'oklch(0.9846 0.0017 247.8389)',
    'muted-foreground': 'oklch(0.5510 0.0234 264.3637)',
    accent: 'oklch(0.9119 0.0222 243.8174)',
    'accent-foreground': 'oklch(0.3791 0.1378 265.5222)',
    destructive: 'oklch(0.6368 0.2078 25.3313)',
    'destructive-foreground': 'oklch(1.0000 0 0)',
    border: 'oklch(0.9022 0.0052 247.8822)',
    input: 'oklch(0.9700 0.0029 264.5420)',
    ring: 'oklch(0.6397 0.1720 36.4421)',
    'chart-1': 'oklch(0.7156 0.0605 248.6845)',
    'chart-2': 'oklch(0.7875 0.0917 35.9616)',
    'chart-3': 'oklch(0.5778 0.0759 254.1573)',
    'chart-4': 'oklch(0.5016 0.0849 259.4902)',
    'chart-5': 'oklch(0.4241 0.0952 264.0306)',
    sidebar: 'oklch(0.9030 0.0046 258.3257)',
    'sidebar-foreground': 'oklch(0.3211 0 0)',
    'sidebar-primary': 'oklch(0.6397 0.1720 36.4421)',
    'sidebar-primary-foreground': 'oklch(1.0000 0 0)',
    'sidebar-accent': 'oklch(0.9119 0.0222 243.8174)',
    'sidebar-accent-foreground': 'oklch(0.3791 0.1378 265.5222)',
    'sidebar-border': 'oklch(0.9276 0.0058 264.5313)',
    'sidebar-ring': 'oklch(0.6397 0.1720 36.4421)',
  },
  dark: {
    background: 'oklch(0.2598 0.0306 262.6666)',
    foreground: 'oklch(0.9219 0 0)',
    card: 'oklch(0.3106 0.0301 268.6365)',
    'card-foreground': 'oklch(0.9219 0 0)',
    popover: 'oklch(0.2900 0.0249 268.3986)',
    'popover-foreground': 'oklch(0.9219 0 0)',
    primary: 'oklch(0.6397 0.1720 36.4421)',
    'primary-foreground': 'oklch(1.0000 0 0)',
    secondary: 'oklch(0.3095 0.0266 266.7132)',
    'secondary-foreground': 'oklch(0.9219 0 0)',
    muted: 'oklch(0.3095 0.0266 266.7132)',
    'muted-foreground': 'oklch(0.7155 0 0)',
    accent: 'oklch(0.3380 0.0589 267.5867)',
    'accent-foreground': 'oklch(0.8823 0.0571 254.1284)',
    destructive: 'oklch(0.6368 0.2078 25.3313)',
    'destructive-foreground': 'oklch(1.0000 0 0)',
    border: 'oklch(0.3843 0.0301 269.7337)',
    input: 'oklch(0.3843 0.0301 269.7337)',
    ring: 'oklch(0.6397 0.1720 36.4421)',
    'chart-1': 'oklch(0.7156 0.0605 248.6845)',
    'chart-2': 'oklch(0.7693 0.0876 34.1875)',
    'chart-3': 'oklch(0.5778 0.0759 254.1573)',
    'chart-4': 'oklch(0.5016 0.0849 259.4902)',
    'chart-5': 'oklch(0.4241 0.0952 264.0306)',
    sidebar: 'oklch(0.3100 0.0283 267.7408)',
    'sidebar-foreground': 'oklch(0.9219 0 0)',
    'sidebar-primary': 'oklch(0.6397 0.1720 36.4421)',
    'sidebar-primary-foreground': 'oklch(1.0000 0 0)',
    'sidebar-accent': 'oklch(0.3380 0.0589 267.5867)',
    'sidebar-accent-foreground': 'oklch(0.8823 0.0571 254.1284)',
    'sidebar-border': 'oklch(0.3843 0.0301 269.7337)',
    'sidebar-ring': 'oklch(0.6397 0.1720 36.4421)',
  },
};

// Supabase 主题（绿色调）
export const supabaseTheme: Theme = {
  name: 'supabase',
  label: 'Supabase',
  light: {
    background: 'oklch(0.9911 0 0)',
    foreground: 'oklch(0.2046 0 0)',
    card: 'oklch(0.9911 0 0)',
    'card-foreground': 'oklch(0.2046 0 0)',
    popover: 'oklch(0.9911 0 0)',
    'popover-foreground': 'oklch(0.4386 0 0)',
    primary: 'oklch(0.8348 0.1302 160.9080)',
    'primary-foreground': 'oklch(0.2626 0.0147 166.4589)',
    secondary: 'oklch(0.9940 0 0)',
    'secondary-foreground': 'oklch(0.2046 0 0)',
    muted: 'oklch(0.9461 0 0)',
    'muted-foreground': 'oklch(0.2435 0 0)',
    accent: 'oklch(0.9461 0 0)',
    'accent-foreground': 'oklch(0.2435 0 0)',
    destructive: 'oklch(0.5523 0.1927 32.7272)',
    'destructive-foreground': 'oklch(0.9934 0.0032 17.2118)',
    border: 'oklch(0.9037 0 0)',
    input: 'oklch(0.9731 0 0)',
    ring: 'oklch(0.8348 0.1302 160.9080)',
    'chart-1': 'oklch(0.8348 0.1302 160.9080)',
    'chart-2': 'oklch(0.6231 0.1880 259.8145)',
    'chart-3': 'oklch(0.6056 0.2189 292.7172)',
    'chart-4': 'oklch(0.7686 0.1647 70.0804)',
    'chart-5': 'oklch(0.6959 0.1491 162.4796)',
    sidebar: 'oklch(0.9911 0 0)',
    'sidebar-foreground': 'oklch(0.5452 0 0)',
    'sidebar-primary': 'oklch(0.8348 0.1302 160.9080)',
    'sidebar-primary-foreground': 'oklch(0.2626 0.0147 166.4589)',
    'sidebar-accent': 'oklch(0.9461 0 0)',
    'sidebar-accent-foreground': 'oklch(0.2435 0 0)',
    'sidebar-border': 'oklch(0.9037 0 0)',
    'sidebar-ring': 'oklch(0.8348 0.1302 160.9080)',
  },
  dark: {
    background: 'oklch(0.1822 0 0)',
    foreground: 'oklch(0.9288 0.0126 255.5078)',
    card: 'oklch(0.2046 0 0)',
    'card-foreground': 'oklch(0.9288 0.0126 255.5078)',
    popover: 'oklch(0.2603 0 0)',
    'popover-foreground': 'oklch(0.7348 0 0)',
    primary: 'oklch(0.4365 0.1044 156.7556)',
    'primary-foreground': 'oklch(0.9213 0.0135 167.1556)',
    secondary: 'oklch(0.2603 0 0)',
    'secondary-foreground': 'oklch(0.9851 0 0)',
    muted: 'oklch(0.2393 0 0)',
    'muted-foreground': 'oklch(0.7122 0 0)',
    accent: 'oklch(0.3132 0 0)',
    'accent-foreground': 'oklch(0.9851 0 0)',
    destructive: 'oklch(0.3123 0.0852 29.7877)',
    'destructive-foreground': 'oklch(0.9368 0.0045 34.3092)',
    border: 'oklch(0.2809 0 0)',
    input: 'oklch(0.2603 0 0)',
    ring: 'oklch(0.8003 0.1821 151.7110)',
    'chart-1': 'oklch(0.8003 0.1821 151.7110)',
    'chart-2': 'oklch(0.7137 0.1434 254.6240)',
    'chart-3': 'oklch(0.7090 0.1592 293.5412)',
    'chart-4': 'oklch(0.8369 0.1644 84.4286)',
    'chart-5': 'oklch(0.7845 0.1325 181.9120)',
    sidebar: 'oklch(0.1822 0 0)',
    'sidebar-foreground': 'oklch(0.6301 0 0)',
    'sidebar-primary': 'oklch(0.4365 0.1044 156.7556)',
    'sidebar-primary-foreground': 'oklch(0.9213 0.0135 167.1556)',
    'sidebar-accent': 'oklch(0.3132 0 0)',
    'sidebar-accent-foreground': 'oklch(0.9851 0 0)',
    'sidebar-border': 'oklch(0.2809 0 0)',
    'sidebar-ring': 'oklch(0.8003 0.1821 151.7110)',
  },
};


// Twitter 主题（蓝色调）
export const twitterTheme: Theme = {
  name: 'twitter',
  label: 'Twitter',
  light: {
    background: 'oklch(1.0000 0 0)',
    foreground: 'oklch(0.1451 0.0040 285.8230)',
    card: 'oklch(1.0000 0 0)',
    'card-foreground': 'oklch(0.1451 0.0040 285.8230)',
    popover: 'oklch(1.0000 0 0)',
    'popover-foreground': 'oklch(0.1451 0.0040 285.8230)',
    primary: 'oklch(0.6231 0.1880 259.8145)',
    'primary-foreground': 'oklch(1.0000 0 0)',
    secondary: 'oklch(0.9670 0.0029 264.5419)',
    'secondary-foreground': 'oklch(0.1451 0.0040 285.8230)',
    muted: 'oklch(0.9670 0.0029 264.5419)',
    'muted-foreground': 'oklch(0.5510 0.0234 264.3637)',
    accent: 'oklch(0.9119 0.0222 243.8174)',
    'accent-foreground': 'oklch(0.1451 0.0040 285.8230)',
    destructive: 'oklch(0.6368 0.2078 25.3313)',
    'destructive-foreground': 'oklch(1.0000 0 0)',
    border: 'oklch(0.9022 0.0052 247.8822)',
    input: 'oklch(0.9700 0.0029 264.5420)',
    ring: 'oklch(0.6231 0.1880 259.8145)',
    'chart-1': 'oklch(0.6231 0.1880 259.8145)',
    'chart-2': 'oklch(0.7875 0.0917 35.9616)',
    'chart-3': 'oklch(0.5778 0.0759 254.1573)',
    'chart-4': 'oklch(0.5016 0.0849 259.4902)',
    'chart-5': 'oklch(0.4241 0.0952 264.0306)',
    sidebar: 'oklch(0.9850 0 0)',
    'sidebar-foreground': 'oklch(0.1451 0.0040 285.8230)',
    'sidebar-primary': 'oklch(0.6231 0.1880 259.8145)',
    'sidebar-primary-foreground': 'oklch(1.0000 0 0)',
    'sidebar-accent': 'oklch(0.9119 0.0222 243.8174)',
    'sidebar-accent-foreground': 'oklch(0.1451 0.0040 285.8230)',
    'sidebar-border': 'oklch(0.9276 0.0058 264.5313)',
    'sidebar-ring': 'oklch(0.6231 0.1880 259.8145)',
  },
  dark: {
    background: 'oklch(0.1451 0.0040 285.8230)',
    foreground: 'oklch(0.9850 0 0)',
    card: 'oklch(0.2046 0.0040 285.8230)',
    'card-foreground': 'oklch(0.9850 0 0)',
    popover: 'oklch(0.2046 0.0040 285.8230)',
    'popover-foreground': 'oklch(0.9850 0 0)',
    primary: 'oklch(0.6231 0.1880 259.8145)',
    'primary-foreground': 'oklch(1.0000 0 0)',
    secondary: 'oklch(0.2809 0.0040 285.8230)',
    'secondary-foreground': 'oklch(0.9850 0 0)',
    muted: 'oklch(0.2809 0.0040 285.8230)',
    'muted-foreground': 'oklch(0.7155 0 0)',
    accent: 'oklch(0.3380 0.0589 267.5867)',
    'accent-foreground': 'oklch(0.9850 0 0)',
    destructive: 'oklch(0.6368 0.2078 25.3313)',
    'destructive-foreground': 'oklch(1.0000 0 0)',
    border: 'oklch(0.3843 0.0040 285.8230)',
    input: 'oklch(0.3843 0.0040 285.8230)',
    ring: 'oklch(0.6231 0.1880 259.8145)',
    'chart-1': 'oklch(0.6231 0.1880 259.8145)',
    'chart-2': 'oklch(0.7693 0.0876 34.1875)',
    'chart-3': 'oklch(0.5778 0.0759 254.1573)',
    'chart-4': 'oklch(0.5016 0.0849 259.4902)',
    'chart-5': 'oklch(0.4241 0.0952 264.0306)',
    sidebar: 'oklch(0.1451 0.0040 285.8230)',
    'sidebar-foreground': 'oklch(0.9850 0 0)',
    'sidebar-primary': 'oklch(0.6231 0.1880 259.8145)',
    'sidebar-primary-foreground': 'oklch(1.0000 0 0)',
    'sidebar-accent': 'oklch(0.3380 0.0589 267.5867)',
    'sidebar-accent-foreground': 'oklch(0.9850 0 0)',
    'sidebar-border': 'oklch(0.3843 0.0040 285.8230)',
    'sidebar-ring': 'oklch(0.6231 0.1880 259.8145)',
  },
};

// 紫色主题
export const violetTheme: Theme = {
  name: 'violet',
  label: '紫罗兰',
  light: {
    background: 'oklch(0.9911 0 0)',
    foreground: 'oklch(0.2046 0 0)',
    card: 'oklch(1.0000 0 0)',
    'card-foreground': 'oklch(0.2046 0 0)',
    popover: 'oklch(1.0000 0 0)',
    'popover-foreground': 'oklch(0.2046 0 0)',
    primary: 'oklch(0.6056 0.2189 292.7172)',
    'primary-foreground': 'oklch(1.0000 0 0)',
    secondary: 'oklch(0.9670 0.0029 264.5419)',
    'secondary-foreground': 'oklch(0.2046 0 0)',
    muted: 'oklch(0.9461 0 0)',
    'muted-foreground': 'oklch(0.5510 0.0234 264.3637)',
    accent: 'oklch(0.9119 0.0422 293.8174)',
    'accent-foreground': 'oklch(0.2046 0 0)',
    destructive: 'oklch(0.5523 0.1927 32.7272)',
    'destructive-foreground': 'oklch(1.0000 0 0)',
    border: 'oklch(0.9037 0 0)',
    input: 'oklch(0.9731 0 0)',
    ring: 'oklch(0.6056 0.2189 292.7172)',
    'chart-1': 'oklch(0.6056 0.2189 292.7172)',
    'chart-2': 'oklch(0.6231 0.1880 259.8145)',
    'chart-3': 'oklch(0.8348 0.1302 160.9080)',
    'chart-4': 'oklch(0.7686 0.1647 70.0804)',
    'chart-5': 'oklch(0.6959 0.1491 162.4796)',
    sidebar: 'oklch(0.9850 0 0)',
    'sidebar-foreground': 'oklch(0.2046 0 0)',
    'sidebar-primary': 'oklch(0.6056 0.2189 292.7172)',
    'sidebar-primary-foreground': 'oklch(1.0000 0 0)',
    'sidebar-accent': 'oklch(0.9119 0.0422 293.8174)',
    'sidebar-accent-foreground': 'oklch(0.2046 0 0)',
    'sidebar-border': 'oklch(0.9037 0 0)',
    'sidebar-ring': 'oklch(0.6056 0.2189 292.7172)',
  },
  dark: {
    background: 'oklch(0.1822 0 0)',
    foreground: 'oklch(0.9850 0 0)',
    card: 'oklch(0.2046 0 0)',
    'card-foreground': 'oklch(0.9850 0 0)',
    popover: 'oklch(0.2603 0 0)',
    'popover-foreground': 'oklch(0.9850 0 0)',
    primary: 'oklch(0.7090 0.1592 293.5412)',
    'primary-foreground': 'oklch(0.1822 0 0)',
    secondary: 'oklch(0.2603 0 0)',
    'secondary-foreground': 'oklch(0.9851 0 0)',
    muted: 'oklch(0.2393 0 0)',
    'muted-foreground': 'oklch(0.7122 0 0)',
    accent: 'oklch(0.3132 0.0422 293.8174)',
    'accent-foreground': 'oklch(0.9851 0 0)',
    destructive: 'oklch(0.6368 0.2078 25.3313)',
    'destructive-foreground': 'oklch(1.0000 0 0)',
    border: 'oklch(0.2809 0 0)',
    input: 'oklch(0.2603 0 0)',
    ring: 'oklch(0.7090 0.1592 293.5412)',
    'chart-1': 'oklch(0.7090 0.1592 293.5412)',
    'chart-2': 'oklch(0.7137 0.1434 254.6240)',
    'chart-3': 'oklch(0.8003 0.1821 151.7110)',
    'chart-4': 'oklch(0.8369 0.1644 84.4286)',
    'chart-5': 'oklch(0.7845 0.1325 181.9120)',
    sidebar: 'oklch(0.1822 0 0)',
    'sidebar-foreground': 'oklch(0.9850 0 0)',
    'sidebar-primary': 'oklch(0.7090 0.1592 293.5412)',
    'sidebar-primary-foreground': 'oklch(0.1822 0 0)',
    'sidebar-accent': 'oklch(0.3132 0.0422 293.8174)',
    'sidebar-accent-foreground': 'oklch(0.9851 0 0)',
    'sidebar-border': 'oklch(0.2809 0 0)',
    'sidebar-ring': 'oklch(0.7090 0.1592 293.5412)',
  },
};

// 所有可用主题
export const themes: Theme[] = [
  defaultTheme,
  tangerineTheme,
  supabaseTheme,
  twitterTheme,
  violetTheme,
];

// 根据名称获取主题
export function getThemeByName(name: string): Theme {
  return themes.find((t) => t.name === name) || defaultTheme;
}

// 应用主题到 DOM
export function applyTheme(theme: Theme, mode: 'light' | 'dark'): void {
  const colors = mode === 'dark' ? theme.dark : theme.light;
  const root = document.documentElement;

  // 设置所有 CSS 变量
  Object.entries(colors).forEach(([key, value]) => {
    if (value) {
      root.style.setProperty(`--${key}`, value);
    }
  });
}

// 获取系统主题偏好
export function getSystemTheme(): 'light' | 'dark' {
  if (typeof window !== 'undefined') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }
  return 'light';
}

// 存储键名
export const THEME_STORAGE_KEY = 'fusionmail_color_theme';
export const MODE_STORAGE_KEY = 'fusionmail_theme_mode';
