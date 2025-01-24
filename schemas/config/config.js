export const autocertExamples = [
    { provider: "local" },
    {
        provider: "cloudflare",
        email: "abc@gmail",
        domains: ["example.com"],
        options: { auth_token: "c1234565789-abcdefghijklmnopqrst" },
    },
    {
        provider: "clouddns",
        email: "abc@gmail",
        domains: ["example.com"],
        options: {
            client_id: "c1234565789",
            email: "abc@gmail",
            password: "password",
        },
    },
];
export const matchDomainsExamples = ["example.com", "*.example.com"];
