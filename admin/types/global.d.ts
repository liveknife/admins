import type { ECharts } from "echarts";
import type { TableColumns } from "@pureadmin/table";

/**
 * 鍏ㄥ眬绫诲瀷澹版槑锛屾棤闇€寮曞叆鐩存帴鍦?`.vue` 銆乣.ts` 銆乣.tsx` 鏂囦欢浣跨敤鍗冲彲鑾峰緱绫诲瀷鎻愮ず
 */
declare global {
  /**
   * 骞冲彴鐨勫悕绉般€佺増鏈€佽繍琛屾墍闇€鐨刞node`鍜宍pnpm`鐗堟湰銆佷緷璧栥€佹渶鍚庢瀯寤烘椂闂寸殑绫诲瀷鎻愮ず
   */
  const __APP_INFO__: {
    pkg: {
      name: string;
      version: string;
      engines: {
        node: string;
        pnpm: string;
      };
      dependencies: Recordable<string>;
      devDependencies: Recordable<string>;
    };
    lastBuildTime: string;
  };

  /**
   * Window 鐨勭被鍨嬫彁绀?
   */
  interface Window {
    // Global vue app instance
    __APP__: App<Element>;
    webkitCancelAnimationFrame: (handle: number) => void;
    mozCancelAnimationFrame: (handle: number) => void;
    oCancelAnimationFrame: (handle: number) => void;
    msCancelAnimationFrame: (handle: number) => void;
    webkitRequestAnimationFrame: (callback: FrameRequestCallback) => number;
    mozRequestAnimationFrame: (callback: FrameRequestCallback) => number;
    oRequestAnimationFrame: (callback: FrameRequestCallback) => number;
    msRequestAnimationFrame: (callback: FrameRequestCallback) => number;
  }

  /**
   * Document 鐨勭被鍨嬫彁绀?
   */
  interface Document {
    webkitFullscreenElement?: Element;
    mozFullScreenElement?: Element;
    msFullscreenElement?: Element;
  }

  /**
   * 鎵撳寘鍘嬬缉鏍煎紡鐨勭被鍨嬪０鏄?
   */
  type ViteCompression =
    | "none"
    | "gzip"
    | "brotli"
    | "both"
    | "gzip-clear"
    | "brotli-clear"
    | "both-clear";

  /**
   * 鍏ㄥ眬鑷畾涔夌幆澧冨彉閲忕殑绫诲瀷澹版槑
   * @see {@link https://pure-admin.cn/pages/config/#%E5%85%B7%E4%BD%93%E9%85%8D%E7%BD%AE}
   */
  interface ViteEnv {
    VITE_PORT: number;
    VITE_PUBLIC_PATH: string;
    VITE_API_BASE_URL: string;
    VITE_API_PROXY_TARGET: string;
    VITE_ROUTER_HISTORY: string;
    VITE_CDN: boolean;
    VITE_HIDE_HOME: string;
    VITE_COMPRESSION: ViteCompression;
  }

  /**
   *  缁ф壙 `@pureadmin/table` 鐨?`TableColumns` 锛屾柟渚垮叏灞€鐩存帴璋冪敤
   */
  type TableColumnList = Array<TableColumns>;

  /**
   * 瀵瑰簲 `public/platform-config.json` 鏂囦欢鐨勭被鍨嬪０鏄?
   * @see {@link https://pure-admin.cn/pages/config/#platform-config-json}
   */
  interface PlatformConfigs {
    Version?: string;
    Title?: string;
    FixedHeader?: boolean;
    HiddenSideBar?: boolean;
    MultiTagsCache?: boolean;
    MaxTagsLevel?: number;
    KeepAlive?: boolean;
    Locale?: string;
    Layout?: string;
    Theme?: string;
    DarkMode?: boolean;
    OverallStyle?: string;
    Grey?: boolean;
    Weak?: boolean;
    HideTabs?: boolean;
    HideFooter?: boolean;
    Stretch?: boolean | number;
    SidebarStatus?: boolean;
    EpThemeColor?: string;
    ShowLogo?: boolean;
    ShowModel?: string;
    MenuArrowIconNoTransition?: boolean;
    CachingAsyncRoutes?: boolean;
    TooltipEffect?: Effect;
    ResponsiveStorageNameSpace?: string;
    MenuSearchHistory?: number;
  }

  /**
   * 涓?`PlatformConfigs` 绫诲瀷涓嶅悓锛岃繖閲屾槸缂撳瓨鍒版祻瑙堝櫒鏈湴瀛樺偍鐨勭被鍨嬪０鏄?
   * @see {@link https://pure-admin.cn/pages/config/#platform-config-json}
   */
  interface StorageConfigs {
    version?: string;
    title?: string;
    fixedHeader?: boolean;
    hiddenSideBar?: boolean;
    multiTagsCache?: boolean;
    keepAlive?: boolean;
    locale?: string;
    layout?: string;
    theme?: string;
    darkMode?: boolean;
    grey?: boolean;
    weak?: boolean;
    hideTabs?: boolean;
    hideFooter?: boolean;
    sidebarStatus?: boolean;
    epThemeColor?: string;
    themeColor?: string;
    overallStyle?: string;
    showLogo?: boolean;
    showModel?: string;
    menuSearchHistory?: number;
    username?: string;
  }

  /**
   * `responsive-storage` 鏈湴鍝嶅簲寮?`storage` 鐨勭被鍨嬪０鏄?
   */
  interface ResponsiveStorage {
    locale: {
      locale?: string;
    };
    layout: {
      layout?: string;
      theme?: string;
      darkMode?: boolean;
      sidebarStatus?: boolean;
      epThemeColor?: string;
      themeColor?: string;
      overallStyle?: string;
    };
    configure: {
      grey?: boolean;
      weak?: boolean;
      hideTabs?: boolean;
      hideFooter?: boolean;
      showLogo?: boolean;
      showModel?: string;
      multiTagsCache?: boolean;
      stretch?: boolean | number;
    };
    tags?: Array<any>;
  }

  /**
   * 骞冲彴閲屾墍鏈夌粍浠跺疄渚嬮兘鑳借闂埌鐨勫叏灞€灞炴€у璞＄殑绫诲瀷澹版槑
   */
  interface GlobalPropertiesApi {
    $echarts: ECharts;
    $storage: ResponsiveStorage;
    $config: PlatformConfigs;
  }

  /**
   * 鎵╁睍 `Element`
   */
  interface Element {
    // v-ripple 浣滅敤浜?src/directives/ripple/index.ts 鏂囦欢
    _ripple?: {
      enabled?: boolean;
      centered?: boolean;
      class?: string;
      circle?: boolean;
      touched?: boolean;
    };
  }
}



