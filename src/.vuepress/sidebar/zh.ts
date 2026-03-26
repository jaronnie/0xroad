// @ts-ignore
import { sidebar } from "vuepress-theme-hope";

export const zhSidebarConfig = sidebar({
    "/": [
        "",
        {
            text: "Solidity",
            icon: "/solidity.svg",
            prefix: "solidity/",
            children: "structure",
            collapsible: true,
            expanded: true,
        },
        {
            text: "ERC 协议",
            icon: "/ERC.svg",
            prefix: "ERC_protocol/",
            children: "structure",
            collapsible: true,
            expanded: true,
        },
        {
            text: "Uniswap",
            icon: "/uniswap-v2.svg",
            prefix: "Uniswap/",
            children: "structure",
            collapsible: true,
            expanded: true,
        },
    ]
});
