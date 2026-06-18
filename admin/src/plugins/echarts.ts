import type { App } from "vue";
import * as echarts from "echarts/core";
import { PieChart, BarChart, LineChart, GaugeChart } from "echarts/charts";
import { CanvasRenderer, SVGRenderer } from "echarts/renderers";
import {
  GridComponent,
  TitleComponent,
  PolarComponent,
  LegendComponent,
  GraphicComponent,
  ToolboxComponent,
  TooltipComponent,
  DataZoomComponent,
  VisualMapComponent
} from "echarts/components";

const { use } = echarts;

use([
  PieChart,
  BarChart,
  LineChart,
  GaugeChart,
  CanvasRenderer,
  SVGRenderer,
  GridComponent,
  TitleComponent,
  PolarComponent,
  LegendComponent,
  GraphicComponent,
  ToolboxComponent,
  TooltipComponent,
  DataZoomComponent,
  VisualMapComponent
]);

/** 按需引入 ECharts，并挂载到 Vue 全局属性。 */
export function useEcharts(app: App) {
  app.config.globalProperties.$echarts = echarts;
}

export default echarts;
