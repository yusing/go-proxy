import { accessLogExamples } from "../config/entrypoint";
export const PROXY_SCHEMES = ["http", "https"];
export const STREAM_SCHEMES = ["tcp", "udp"];
export const homepageExamples = [
    {
        name: "Sonarr",
        icon: "png/sonarr.png",
        category: "Arr suite",
    },
    {
        name: "App",
        icon: "@target/favicon.ico",
    },
];
export const loadBalanceExamples = [
    {
        link: "flaresolverr",
        mode: "round_robin",
    },
    {
        link: "service.domain.com",
        mode: "ip_hash",
        config: {
            header: "X-Real-IP",
        },
    },
];
export { accessLogExamples };
