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
            collapsible: true,
            prefix: "Uniswap/",
            expanded: true,
            children: [
                {
                    text: "V2",
                    icon: "/uniswap-v2.svg",
                    prefix: "V2/",
                    children: "structure",
                    collapsible: true,
                    expanded: false,
                },
                {
                    text: "V3",
                    icon: "/uniswap-v3.svg",
                    prefix: "V3/",
                    children: "structure",
                    collapsible: true,
                    expanded: false,
                },
                {
                    text: "V4",
                    prefix: "V4/",
                    icon: "/uniswap-v3.svg",
                    children: "structure",
                    collapsible: true,
                    expanded: false,
                },
            ],
        },
    ]
});
