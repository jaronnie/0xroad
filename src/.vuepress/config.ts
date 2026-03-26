// @ts-ignore
import { defineUserConfig } from "vuepress";
import theme from "./theme.js";

export default defineUserConfig({
  base: "/",

  locales: {
    "/": {
      lang: "zh-CN",
      title: "0xroad",
      description: "The road to 0x world",
    },
    "/en/": {
      lang: "en-US",
      title: "0xroad",
      description: "The road to 0x world",
    },
  },

  theme,

  // 和 PWA 一起启用
  // shouldPrefetch: false,
});
