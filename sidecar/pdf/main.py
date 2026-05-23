from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import base64
from io import BytesIO
from pypdf import PdfWriter, PdfReader
from reportlab.pdfgen import canvas
from reportlab.lib.pagesizes import A4

app = FastAPI(title="onefacture-pdf-sidecar")

class GeneratePDFRequest(BaseModel):
    invoice_number: str
    profile: str
    xml_base64: str
    seller_name: str = "Seller"
    buyer_name: str = "Buyer"
    total_ht: float = 0.0
    total_ttc: float = 0.0

@app.post("/generate")
async def generate_facturx_pdf(req: GeneratePDFRequest):
    try:
        xml_bytes = base64.b64decode(req.xml_base64)

        # Create a minimal visual PDF
        buffer = BytesIO()
        c = canvas.Canvas(buffer, pagesize=A4)
        width, height = A4

        c.setFont("Helvetica-Bold", 16)
        c.drawString(50, height - 50, f"Facture {req.invoice_number}")
        c.setFont("Helvetica", 12)
        c.drawString(50, height - 80, f"Profil: {req.profile}")
        c.drawString(50, height - 100, f"Vendeur: {req.seller_name}")
        c.drawString(50, height - 120, f"Acheteur: {req.buyer_name}")

        # Simple line items placeholder
        c.setFont("Helvetica-Bold", 11)
        c.drawString(50, height - 170, "Lignes de facture (exemple)")
        c.setFont("Helvetica", 10)
        c.drawString(50, height - 190, "1. Prestation de service ................ 1 200,00 €")
        c.drawString(50, height - 205, "2. Frais de dossier ......................  80,00 €")
        c.drawString(50, height - 230, f"Total HT: {req.total_ht:.2f} € | Total TTC: {req.total_ttc:.2f} €")

        c.drawString(50, height - 270, "[Factur-X XML attaché - généré par sidecar]")

        # Basic PDF/A-3 metadata (XMP-like via reportlab)
        c.setTitle(f"Facture {req.invoice_number}")
        c.setAuthor("onefacture")
        c.setSubject("Factur-X / EN 16931")
        c.setCreator("onefacture PDF sidecar")
        c.setKeywords("Factur-X, EN16931, CII")
        c.save()

        # Create PDF/A-3 like container and attach XML
        buffer.seek(0)
        reader = PdfReader(buffer)
        writer = PdfWriter()

        for page in reader.pages:
            writer.add_page(page)

        # Attach the XML as embedded file (Factur-X style)
        writer.add_attachment("factur-x.xml", xml_bytes)

        # Set PDF/A-3 metadata (XMP)
        writer.add_metadata({
            "/Title": f"Facture {req.invoice_number}",
            "/Author": "onefacture",
            "/Subject": "Factur-X / EN 16931",
            "/Keywords": "Factur-X, EN16931, CII",
            "/Creator": "onefacture PDF sidecar",
            "/Producer": "onefacture + pypdf",
            "/pdfaid:part": "3",
            "/pdfaid:conformance": "A",
        })

        output = BytesIO()
        writer.write(output)
        output.seek(0)

        pdf_base64 = base64.b64encode(output.read()).decode()

        return {
            "pdf_base64": pdf_base64,
            "filename": f"facture-{req.invoice_number}.pdf",
            "note": "Minimal PDF/A-3 container with embedded CII XML (sidecar prototype)"
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/healthz")
async def health():
    return {"status": "ok", "service": "pdf-sidecar"}
