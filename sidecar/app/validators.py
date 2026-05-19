"""Validation primitives used by the sidecar FastAPI app."""

from __future__ import annotations

import logging
from dataclasses import dataclass, field
from pathlib import Path
from typing import Iterable

from lxml import etree

logger = logging.getLogger(__name__)

Severity = str  # "error" | "warning"


@dataclass
class Finding:
    layer: str
    code: str
    severity: Severity
    message: str
    path: str = ""

    def to_dict(self) -> dict:
        return {
            "layer": self.layer,
            "code": self.code,
            "severity": self.severity,
            "message": self.message,
            "path": self.path,
        }


@dataclass
class ValidationReport:
    valid: bool
    profile: str
    findings: list[Finding] = field(default_factory=list)

    def to_dict(self) -> dict:
        return {
            "valid": self.valid,
            "profile": self.profile,
            "findings": [f.to_dict() for f in self.findings],
        }


class FacturXValidator:
    """Runs XSD + compiled Schematron (XSL) validation against a CII XML."""

    def __init__(self, specs_root: Path) -> None:
        self.specs_root = specs_root
        self._xsd_schema = self._load_xsd()
        self._schematron_xsl = self._load_schematron()

    def _load_xsd(self) -> etree.XMLSchema | None:
        path = self.specs_root / "xsd" / "Factur-X_1.08_EN16931.xsd"
        if not path.exists():
            logger.warning("XSD not found at %s", path)
            return None
        try:
            doc = etree.parse(str(path))
            return etree.XMLSchema(doc)
        except etree.XMLSchemaParseError as e:
            logger.error("XSD parse error: %s", e)
            return None

    def _load_schematron(self) -> etree.XSLT | None:
        path = self.specs_root / "schematron" / "Factur-X_1.08_EN16931-compiled.xsl"
        if not path.exists():
            logger.warning("Schematron XSL not found at %s", path)
            return None
        try:
            doc = etree.parse(str(path))
            return etree.XSLT(doc)
        except (etree.XMLSyntaxError, etree.XSLTParseError) as e:
            logger.error("Schematron load error: %s", e)
            return None

    def validate_xml(self, raw: bytes, profile: str = "EN16931") -> ValidationReport:
        findings: list[Finding] = []
        try:
            doc = etree.fromstring(raw)
        except etree.XMLSyntaxError as e:
            findings.append(Finding("syntax", "XML_PARSE", "error", str(e)))
            return ValidationReport(valid=False, profile=profile, findings=findings)

        # XSD (layer 3)
        if self._xsd_schema is not None:
            if not self._xsd_schema.validate(doc):
                for err in self._xsd_schema.error_log:
                    findings.append(
                        Finding(
                            layer="xsd",
                            code="XSD_VIOLATION",
                            severity="error",
                            message=err.message,
                            path=err.path or "",
                        )
                    )
        else:
            findings.append(Finding("xsd", "XSD_UNAVAILABLE", "warning", "XSD schema not loaded"))

        # Schematron (layers 4-5)
        if self._schematron_xsl is not None:
            try:
                report = self._schematron_xsl(doc)
                findings.extend(self._parse_svrl(report))
            except etree.XSLTApplyError as e:
                findings.append(Finding("schematron", "SCH_APPLY_ERROR", "warning", str(e)))
        else:
            findings.append(
                Finding("schematron", "SCH_UNAVAILABLE", "warning", "Schematron XSL not loaded")
            )

        valid = not any(f.severity == "error" for f in findings)
        return ValidationReport(valid=valid, profile=profile, findings=findings)

    @staticmethod
    def _parse_svrl(svrl_doc: etree._XSLTResultTree) -> Iterable[Finding]:
        ns = {"svrl": "http://purl.oclc.org/dsdl/svrl"}
        root = svrl_doc.getroot()
        if root is None:
            return []
        out: list[Finding] = []
        for el in root.findall(".//svrl:failed-assert", ns):
            out.append(
                Finding(
                    layer="schematron",
                    code=el.get("id", "SCH"),
                    severity="error" if el.get("flag", "fatal") != "warning" else "warning",
                    message=(el.findtext("svrl:text", default="", namespaces=ns) or "").strip(),
                    path=el.get("location", ""),
                )
            )
        for el in root.findall(".//svrl:successful-report", ns):
            out.append(
                Finding(
                    layer="schematron",
                    code=el.get("id", "SCH"),
                    severity="warning",
                    message=(el.findtext("svrl:text", default="", namespaces=ns) or "").strip(),
                    path=el.get("location", ""),
                )
            )
        return out


class PDFExtractor:
    """Extracts the embedded factur-x.xml from a PDF/A-3 byte stream."""

    EMBEDDED_FILENAMES = (b"factur-x.xml", b"zugferd-invoice.xml", b"ZUGFeRD-invoice.xml")

    def extract(self, raw: bytes) -> bytes | None:
        # Light heuristic search; for real extraction use pikepdf in production.
        for needle in self.EMBEDDED_FILENAMES:
            idx = raw.find(needle)
            if idx == -1:
                continue
            # Find the next 'stream' marker after the filespec.
            stream_idx = raw.find(b"stream\n", idx)
            if stream_idx == -1:
                continue
            end_idx = raw.find(b"\nendstream", stream_idx)
            if end_idx == -1:
                continue
            return raw[stream_idx + len(b"stream\n") : end_idx]
        return None
