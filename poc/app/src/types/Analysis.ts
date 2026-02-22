
export type Analysis = {
    is_malicious: boolean;
    confidence: number;
    justification: string;
    indicators: string[]
};