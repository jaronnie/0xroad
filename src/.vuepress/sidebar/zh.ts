// @ts-ignore
import { sidebar } from "vuepress-theme-hope";

export const zhSidebarConfig = sidebar({
    "/": [
        "",
        {
            text: "ERC 协议",
            icon: "/ERC.svg",
            prefix: "ERC_protocol/",
            children: "structure",
            collapsible: true,
            expanded: true,
        },
    ]
});
