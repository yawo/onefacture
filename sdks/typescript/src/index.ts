export type ClientOptions = {
  apiKey: string;
  baseUrl?: string;
};

export class OnefactureClient {
  private readonly apiKey: string;
  private readonly baseUrl: string;

  constructor(options: ClientOptions) {
    this.apiKey = options.apiKey;
    this.baseUrl = (options.baseUrl ?? "https://api.onefacture.io").replace(/\/$/, "");
  }

  async createInvoice(invoice: unknown, options: { submit?: boolean; idempotencyKey?: string } = {}) {
    const response = await fetch(`${this.baseUrl}/v1/invoices?submit=${options.submit === true}`, {
      method: "POST",
      headers: this.headers(options.idempotencyKey),
      body: JSON.stringify(invoice),
    });
    if (!response.ok) throw new Error(await response.text());
    return response.json();
  }

  async retryInvoice(invoiceId: string, resolutionHint = "") {
    const response = await fetch(`${this.baseUrl}/v1/invoices/${invoiceId}/retry`, {
      method: "POST",
      headers: this.headers(),
      body: JSON.stringify({ resolution_hint: resolutionHint }),
    });
    if (!response.ok) throw new Error(await response.text());
    return response.json();
  }

  private headers(idempotencyKey: string = crypto.randomUUID()) {
    return {
      "Content-Type": "application/json",
      "Idempotency-Key": idempotencyKey,
      "X-API-Key": this.apiKey,
    };
  }
}
