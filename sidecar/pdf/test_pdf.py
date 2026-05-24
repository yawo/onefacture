import pytest
from fastapi.testclient import TestClient
from main import app
import base64
from io import BytesIO
from pypdf import PdfReader

client = TestClient(app)

def test_health():
    resp = client.get("/healthz")
    assert resp.status_code == 200
    assert resp.json()["status"] == "ok"

def test_generate_basic():
    xml = b'<?xml version="1.0"?><CrossIndustryInvoice></CrossIndustryInvoice>'
    payload = {
        "invoice_number": "TEST-001",
        "profile": "EN16931",
        "xml_base64": base64.b64encode(xml).decode(),
        "seller_name": "Test Seller",
        "buyer_name": "Test Buyer"
    }
    resp = client.post("/generate", json=payload)
    assert resp.status_code == 200
    data = resp.json()
    assert "pdf_base64" in data
    assert data["filename"].startswith("facture-TEST-001")
    pdf_bytes = base64.b64decode(data["pdf_base64"])
    assert pdf_bytes.startswith(b"%PDF-1.")

    # Verify that the XML is extractible (real Factur-X requirement)
    reader = PdfReader(BytesIO(pdf_bytes))
    attachments = reader.attachments
    assert "factur-x.xml" in attachments
    extracted_xml = attachments["factur-x.xml"][0]
    assert b"CrossIndustryInvoice" in extracted_xml or b"<?xml" in extracted_xml
