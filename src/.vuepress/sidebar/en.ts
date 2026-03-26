// @ts-ignore
import { sidebar } from "vuepress-theme-hope";

export const enSidebarConfig = sidebar({
    "/en/": [
        "",
        {
            text: "ERC protocol",
            icon: "/ERC.svg",
            prefix: "ERC_protocol/",
            children: "structure",
            collapsible: true,
            expanded: true,
        },
    ]
});
