export const accessLogExamples = [
    {
        path: "/var/log/access.log",
        format: "combined",
        filters: {
            status_codes: {
                values: ["200-299"],
            },
        },
        fields: {
            headers: {
                default: "keep",
                config: {
                    foo: "redact",
                },
            },
        },
    },
];
export const middlewaresExamples = [
    {
        use: "RedirectHTTP",
    },
    {
        use: "CIDRWhitelist",
        allow: ["127.0.0.1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"],
        status: 403,
        message: "Forbidden",
    },
];
