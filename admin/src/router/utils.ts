import {
  type RouterHistory,
  type RouteRecordRaw,
  type RouteComponent,
  createWebHistory,
  createWebHashHistory
} from "vue-router";
import { router } from "./index";
import { isProxy, toRaw } from "vue";
import { useTimeoutFn } from "@vueuse/core";
import {
  isString,
  cloneDeep,
  isAllEmpty,
  intersection,
  storageLocal,
  isIncludeAllChildren
} from "@pureadmin/utils";
import { getConfig } from "@/config";
import { buildHierarchyTree } from "@/utils/tree";
import { userKey, type DataInfo } from "@/utils/auth";
import { type menuType, routerArrays } from "@/layout/types";
import { useMultiTagsStoreHook } from "@/store/modules/multiTags";
import { usePermissionStoreHook } from "@/store/modules/permission";
const IFrame = () => import("@/layout/frame.vue");
// https://cn.vitejs.dev/guide/features.html#glob-import
const modulesRoutes = import.meta.glob("/src/views/**/*.{vue,tsx}");

// 鍔ㄦ€佽矾鐢?
import { getAsyncRoutes } from "@/api/routes";

function handRank(routeInfo: any) {
  const { name, path, parentId, meta } = routeInfo;
  return isAllEmpty(parentId)
    ? isAllEmpty(meta?.rank) ||
      (meta?.rank === 0 && name !== "Home" && path !== "/")
      ? true
      : false
    : false;
}

/** 鎸夌収璺敱涓璵eta涓嬬殑rank绛夌骇鍗囧簭鏉ユ帓搴忚矾鐢?*/
function ascending(arr: any[]) {
  arr.forEach((v, index) => {
    // 褰搑ank涓嶅瓨鍦ㄦ椂锛屾牴鎹『搴忚嚜鍔ㄥ垱寤猴紝棣栭〉璺敱姘歌繙鍦ㄧ涓€浣?
    if (handRank(v)) v.meta.rank = index + 2;
  });
  return arr.sort(
    (a: { meta: { rank: number } }, b: { meta: { rank: number } }) => {
      return a?.meta.rank - b?.meta.rank;
    }
  );
}

/** 杩囨护meta涓璼howLink涓篺alse鐨勮彍鍗?*/
function filterTree(data: RouteComponent[]) {
  const newTree = cloneDeep(data).filter(
    (v: { meta: { showLink: boolean } }) => v.meta?.showLink !== false
  );
  newTree.forEach(
    (v: { children }) => v.children && (v.children = filterTree(v.children))
  );
  return newTree;
}

/** 杩囨护children闀垮害涓?鐨勭殑鐩綍锛屽綋鐩綍涓嬫病鏈夎彍鍗曟椂锛屼細杩囨护姝ょ洰褰曪紝鐩綍娌℃湁璧嬩簣roles鏉冮檺锛屽綋鐩綍涓嬪彧瑕佹湁涓€涓彍鍗曟湁鏄剧ず鏉冮檺锛岄偅涔堟鐩綍灏变細鏄剧ず */
function filterChildrenTree(data: RouteComponent[]) {
  const newTree = cloneDeep(data).filter((v: any) => v?.children?.length !== 0);
  newTree.forEach(
    (v: { children }) => v.children && (v.children = filterTree(v.children))
  );
  return newTree;
}

/** 鍒ゆ柇涓や釜鏁扮粍褰兼鏄惁瀛樺湪鐩稿悓鍊?*/
function isOneOfArray(a: Array<string>, b: Array<string>) {
  return Array.isArray(a) && Array.isArray(b)
    ? intersection(a, b).length > 0
      ? true
      : false
    : true;
}

/** 浠巐ocalStorage閲屽彇鍑哄綋鍓嶇櫥褰曠敤鎴风殑瑙掕壊roles锛岃繃婊ゆ棤鏉冮檺鐨勮彍鍗?*/
function filterNoPermissionTree(data: RouteComponent[]) {
  const currentRoles =
    storageLocal().getItem<DataInfo<number>>(userKey)?.roles ?? [];
  const newTree = cloneDeep(data).filter((v: any) =>
    isOneOfArray(v.meta?.roles, currentRoles)
  );
  newTree.forEach(
    (v: any) => v.children && (v.children = filterNoPermissionTree(v.children))
  );
  return filterChildrenTree(newTree);
}

/** 閫氳繃鎸囧畾 `key` 鑾峰彇鐖剁骇璺緞闆嗗悎锛岄粯璁?`key` 涓?`path` */
function getParentPaths(value: string, routes: RouteRecordRaw[], key = "path") {
  // 娣卞害閬嶅巻鏌ユ壘
  function dfs(routes: RouteRecordRaw[], value: string, parents: string[]) {
    for (let i = 0; i < routes.length; i++) {
      const item = routes[i];
      // 杩斿洖鐖剁骇path
      if (item[key] === value) return parents;
      // children涓嶅瓨鍦ㄦ垨涓虹┖鍒欎笉閫掑綊
      if (!item.children || !item.children.length) continue;
      // 寰€涓嬫煡鎵炬椂灏嗗綋鍓峱ath鍏ユ爤
      parents.push(item.path);

      if (dfs(item.children, value, parents).length) return parents;
      // 娣卞害閬嶅巻鏌ユ壘鏈壘鍒版椂褰撳墠path 鍑烘爤
      parents.pop();
    }
    // 鏈壘鍒版椂杩斿洖绌烘暟缁?
    return [];
  }

  return dfs(routes, value, []);
}

/** 鏌ユ壘瀵瑰簲 `path` 鐨勮矾鐢变俊鎭?*/
function findRouteByPath(path: string, routes: RouteRecordRaw[]) {
  let res = routes.find((item: { path: string }) => item.path == path);
  if (res) {
    return isProxy(res) ? toRaw(res) : res;
  } else {
    for (let i = 0; i < routes.length; i++) {
      if (
        routes[i].children instanceof Array &&
        routes[i].children.length > 0
      ) {
        res = findRouteByPath(path, routes[i].children);
        if (res) {
          return isProxy(res) ? toRaw(res) : res;
        }
      }
    }
    return null;
  }
}

/** 鍔ㄦ€佽矾鐢辨敞鍐屽畬鎴愬悗锛屽啀娣诲姞鍏ㄥ睆404锛堥〉闈笉瀛樺湪锛夐〉闈紝閬垮厤鍒锋柊鍔ㄦ€佽矾鐢遍〉闈㈡椂璇烦杞埌404椤甸潰 */
function addPathMatch() {
  if (!router.hasRoute("pathMatch")) {
    router.addRoute({
      path: "/:pathMatch(.*)*",
      name: "PageNotFound",
      component: () => import("@/views/error/404.vue"),
      meta: {
        title: "404",
        showLink: false
      }
    });
  }
}

/** 澶勭悊鍔ㄦ€佽矾鐢憋紙鍚庣杩斿洖鐨勮矾鐢憋級 */
function handleAsyncRoutes(routeList) {
  if (routeList.length === 0) {
    usePermissionStoreHook().handleWholeMenus(routeList);
  } else {
    formatFlatteningRoutes(addAsyncRoutes(routeList)).map(
      (v: RouteRecordRaw) => {
        // 闃叉閲嶅娣诲姞璺敱
        if (
          router.options.routes[0].children.findIndex(
            value => value.path === v.path
          ) !== -1
        ) {
          return;
        } else {
          // 鍒囪灏嗚矾鐢眕ush鍒皉outes鍚庤繕闇€瑕佷娇鐢╝ddRoute锛岃繖鏍疯矾鐢辨墠鑳芥甯歌烦杞?
          router.options.routes[0].children.push(v);
          // 鏈€缁堣矾鐢辫繘琛屽崌搴?
          ascending(router.options.routes[0].children);
          if (!router.hasRoute(v?.name)) router.addRoute(v);
          const flattenRouters: any = router
            .getRoutes()
            .find(n => n.path === "/");
          // 淇濇寔router.options.routes[0].children涓巔ath涓?/"鐨刢hildren涓€鑷达紝闃叉鏁版嵁涓嶄竴鑷村鑷村紓甯?
          flattenRouters.children = router.options.routes[0].children;
          router.addRoute(flattenRouters);
        }
      }
    );
    usePermissionStoreHook().handleWholeMenus(routeList);
  }
  if (!useMultiTagsStoreHook().getMultiTagsCache) {
    useMultiTagsStoreHook().handleTags("equal", [
      ...routerArrays,
      ...usePermissionStoreHook().flatteningRoutes.filter(
        v => v?.meta?.fixedTag
      )
    ]);
  }
  addPathMatch();
}

/** 初始化路由 */
function initRouter() {
  if (getConfig()?.CachingAsyncRoutes) {
    // 寮€鍚姩鎬佽矾鐢辩紦瀛樻湰鍦發ocalStorage
    const key = "async-routes";
    const asyncRouteList = storageLocal().getItem(key) as any;
    if (asyncRouteList && asyncRouteList?.length > 0) {
      return new Promise(resolve => {
        handleAsyncRoutes(asyncRouteList);
        resolve(router);
      });
    } else {
      return new Promise(resolve => {
        getAsyncRoutes().then(({ data }) => {
          handleAsyncRoutes(cloneDeep(data));
          storageLocal().setItem(key, data);
          resolve(router);
        });
      });
    }
  } else {
    return new Promise(resolve => {
      getAsyncRoutes().then(({ data }) => {
        handleAsyncRoutes(cloneDeep(data));
        resolve(router);
      });
    });
  }
}

/** * 灏嗗绾у祵濂楄矾鐢卞鐞嗘垚涓€缁存暟缁?
 * @param routesList 浼犲叆璺敱
 * @returns 杩斿洖澶勭悊鍚庣殑涓€缁磋矾鐢? */
function formatFlatteningRoutes(routesList: RouteRecordRaw[]) {
  if (routesList.length === 0) return routesList;
  let hierarchyList = buildHierarchyTree(routesList);
  for (let i = 0; i < hierarchyList.length; i++) {
    if (hierarchyList[i].children) {
      hierarchyList = hierarchyList
        .slice(0, i + 1)
        .concat(hierarchyList[i].children, hierarchyList.slice(i + 1));
    }
  }
  return hierarchyList;
}

/** * 涓€缁存暟缁勫鐞嗘垚澶氱骇宓屽鏁扮粍锛堜笁绾у強浠ヤ笂鐨勮矾鐢卞叏閮ㄦ媿鎴愪簩绾э紝keep-alive 鍙敮鎸佸埌浜岀骇缂撳瓨锛?
 * @param routesList 澶勭悊鍚庣殑涓€缁磋矾鐢辫彍鍗曟暟缁?
 * @returns 杩斿洖灏嗕竴缁存暟缁勯噸鏂板鐞嗘垚瑙勫畾璺敱鐨勬牸寮? */
function formatTwoStageRoutes(routesList: RouteRecordRaw[]) {
  if (routesList.length === 0) return routesList;
  const newRoutesList: RouteRecordRaw[] = [];
  routesList.forEach((v: RouteRecordRaw) => {
    if (v.path === "/") {
      newRoutesList.push({
        component: v.component,
        name: v.name,
        path: v.path,
        redirect: v.redirect,
        meta: v.meta,
        children: []
      });
    } else {
      newRoutesList[0]?.children.push({ ...v });
    }
  });
  return newRoutesList;
}

/** 澶勭悊缂撳瓨璺敱锛堟坊鍔犮€佸垹闄ゃ€佸埛鏂帮級 */
function handleAliveRoute({ name }: ToRouteType, mode?: string) {
  switch (mode) {
    case "add":
      usePermissionStoreHook().cacheOperate({
        mode: "add",
        name
      });
      break;
    case "delete":
      usePermissionStoreHook().cacheOperate({
        mode: "delete",
        name
      });
      break;
    case "refresh":
      usePermissionStoreHook().cacheOperate({
        mode: "refresh",
        name
      });
      break;
    default:
      usePermissionStoreHook().cacheOperate({
        mode: "delete",
        name
      });
      useTimeoutFn(() => {
        usePermissionStoreHook().cacheOperate({
          mode: "add",
          name
        });
      }, 100);
  }
}

/** 杩囨护鍚庣浼犳潵鐨勫姩鎬佽矾鐢?閲嶆柊鐢熸垚瑙勮寖璺敱 */
function addAsyncRoutes(arrRoutes: Array<RouteRecordRaw>) {
  if (!arrRoutes || !arrRoutes.length) return;
  const modulesRoutesKeys = Object.keys(modulesRoutes);
  arrRoutes.forEach((v: RouteRecordRaw) => {
    // 灏哹ackstage灞炴€у姞鍏eta锛屾爣璇嗘璺敱涓哄悗绔繑鍥炶矾鐢?
    v.meta.backstage = true;
    // 鐖剁骇鐨剅edirect灞炴€у彇鍊硷細濡傛灉瀛愮骇瀛樺湪涓旂埗绾х殑redirect灞炴€т笉瀛樺湪锛岄粯璁ゅ彇绗竴涓瓙绾х殑path锛涘鏋滃瓙绾у瓨鍦ㄤ笖鐖剁骇鐨剅edirect灞炴€у瓨鍦紝鍙栧瓨鍦ㄧ殑redirect灞炴€э紝浼氳鐩栭粯璁ゅ€?
    if (v?.children && v.children.length && !v.redirect)
      v.redirect = v.children[0].path;
    // 鐖剁骇鐨刵ame灞炴€у彇鍊硷細濡傛灉瀛愮骇瀛樺湪涓旂埗绾х殑name灞炴€т笉瀛樺湪锛岄粯璁ゅ彇绗竴涓瓙绾х殑name锛涘鏋滃瓙绾у瓨鍦ㄤ笖鐖剁骇鐨刵ame灞炴€у瓨鍦紝鍙栧瓨鍦ㄧ殑name灞炴€э紝浼氳鐩栭粯璁ゅ€硷紙娉ㄦ剰锛氭祴璇曚腑鍙戠幇鐖剁骇鐨刵ame涓嶈兘鍜屽瓙绾ame閲嶅锛屽鏋滈噸澶嶄細閫犳垚閲嶅畾鍚戞棤鏁堬紙璺宠浆404锛夛紝鎵€浠ヨ繖閲岀粰鐖剁骇鐨刵ame璧峰悕鐨勬椂鍊欏悗闈細鑷姩鍔犱笂`Parent`锛岄伩鍏嶉噸澶嶏級
    if (v?.children && v.children.length && !v.name)
      v.name = (v.children[0].name as string) + "Parent";
    if (v.meta?.frameSrc) {
      v.component = IFrame;
    } else {
      // 瀵瑰悗绔紶component缁勪欢璺緞鍜屼笉浼犲仛鍏煎锛堝鏋滃悗绔紶component缁勪欢璺緞锛岄偅涔坧ath鍙互闅忎究鍐欙紝濡傛灉涓嶄紶锛宑omponent缁勪欢璺緞浼氳窡path淇濇寔涓€鑷达級
      const index = v?.component
        ? modulesRoutesKeys.findIndex(ev => ev.includes(v.component as any))
        : modulesRoutesKeys.findIndex(ev => ev.includes(v.path));
      v.component = modulesRoutes[modulesRoutesKeys[index]];
    }
    if (v?.children && v.children.length) {
      addAsyncRoutes(v.children);
    }
  });
  return arrRoutes;
}

/** 鑾峰彇璺敱鍘嗗彶妯″紡 https://next.router.vuejs.org/zh/guide/essentials/history-mode.html */
function getHistoryMode(routerHistory): RouterHistory {
  // len涓? 浠ｈ〃鍙湁鍘嗗彶妯″紡 涓? 浠ｈ〃鍘嗗彶妯″紡涓瓨鍦╞ase鍙傛暟 https://next.router.vuejs.org/zh/api/#%E5%8F%82%E6%95%B0-1
  const historyMode = routerHistory.split(",");
  const leftMode = historyMode[0];
  const rightMode = historyMode[1];
  // no param
  if (historyMode.length === 1) {
    if (leftMode === "hash") {
      return createWebHashHistory("");
    } else if (leftMode === "h5") {
      return createWebHistory("");
    }
  } //has param
  else if (historyMode.length === 2) {
    if (leftMode === "hash") {
      return createWebHashHistory(rightMode);
    } else if (leftMode === "h5") {
      return createWebHistory(rightMode);
    }
  }
}

/** 获取当前页面按钮级别权限 */
function getAuths(): Array<string> {
  return router.currentRoute.value.meta.auths as Array<string>;
}

/** 是否拥有按钮级权限，根据路由 meta.auths 字段判断 */
function hasAuth(value: string | Array<string>): boolean {
  if (!value) return false;
  /** 从当前路由 meta 字段中获取按钮级权限 code */
  const metaAuths = getAuths();
  if (!metaAuths) return false;
  const isAuths = isString(value)
    ? metaAuths.includes(value)
    : isIncludeAllChildren(value, metaAuths);
  return isAuths ? true : false;
}

function handleTopMenu(route) {
  if (route?.children && route.children.length > 1) {
    if (route.redirect) {
      return route.children.filter(cur => cur.path === route.redirect)[0];
    } else {
      return route.children[0];
    }
  } else {
    return route;
  }
}

/** 获取所有菜单中的第一个顶级菜单 */
function getTopMenu(tag = false): menuType {
  const topMenu = handleTopMenu(
    usePermissionStoreHook().wholeMenus[0]?.children[0]
  );
  tag && useMultiTagsStoreHook().handleTags("push", topMenu);
  return topMenu;
}

export {
  hasAuth,
  getAuths,
  ascending,
  filterTree,
  initRouter,
  getTopMenu,
  addPathMatch,
  isOneOfArray,
  getHistoryMode,
  addAsyncRoutes,
  getParentPaths,
  findRouteByPath,
  handleAliveRoute,
  formatTwoStageRoutes,
  formatFlatteningRoutes,
  filterNoPermissionTree
};

