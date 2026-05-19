"""onefacture validation sidecar.

Exposes XSD + Schematron validation of Factur-X / CII XML using lxml.
Layer mapping:
  - /v1/validate/xml -> XSD (layer 3) + Schematron EN16931 + AFNOR (layers 4-5)
  - /v1/extract     -> Extract embedded XML from a Factur-X PDF/A-3 (layers 1-2)

Validation artifacts are loaded once at startup from the repo's docs/specs tree.
"""

from __future__ import annotations

import logging
import os
from pathlib import Path

from fastapi import FastAPI, File, Form, UploadFile
from fastapi.responses import JSONResponse

from .validators import (
    FacturXValidator,
    PDFExtractor,
    ValidationReport,
)

logger = logging.getLogger("onefacture.sidecar")
logging.basicConfig(level=os.getenv("ONEFACTURE_SIDECAR_LOG_LEVEL", "INFO"))

SPECS_ROOT = Path(os.getenv("ONEFACTURE_SPECS_ROOT", "/specs"))
if not SPECS_ROOT.exists():
    # Local dev: fall back to repo path.
    SPECS_ROOT = Path(__file__).resolve().parents[2] / "docs" / "specs" / "validation"

app = FastAPI(title="onefacture sidecar", version="0.1.0")

validator = FacturXValidator(specs_root=SPECS_ROOT)
extractor = PDFExtractor()


@app.get("/healthz")
def healthz() -> dict:
    return {"status": "ok", "specs_root": str(SPECS_ROOT)}


@app.post("/v1/validate/xml")
async def validate_xml(file: UploadFile = File(...), profile: str = Form("EN16931")) -> JSONResponse:
    raw = await file.read()
    report: ValidationReport = validator.validate_xml(raw, profile=profile)
    return JSONResponse(report.to_dict())


@app.post("/v1/extract")
async def extract(file: UploadFile = File(...)) -> JSONResponse:
    raw = await file.read()
    xml_bytes = extractor.extract(raw)
    if xml_bytes is None:
        return JSONResponse({"found": False}, status_code=404)
    return JSONResponse({"found": True, "xml": xml_bytes.decode("utf-8", errors="replace")})


@app.post("/v1/package/pdfa3")
async def package_pdfa3(file: UploadFile = File(...), invoice_number: str = Form(...)) -> JSONResponse:
    """Placeholder for PDF/A-3 packaging.

    Real implementation would embed the supplied XML into a PDF/A-3 conformant
    container using pdfa3 + factur-x. We return 501 until the dependency
    is integrated. The Go gateway falls back to a placeholder PDF.
    """
    return JSONResponse(
        {"detail": "PDF/A-3 packaging not yet implemented in sidecar"},
        status_code=501,
    )
