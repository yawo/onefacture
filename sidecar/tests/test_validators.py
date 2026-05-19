from pathlib import Path

from app.validators import FacturXValidator, PDFExtractor


def test_validator_handles_missing_artifacts(tmp_path: Path) -> None:
    v = FacturXValidator(specs_root=tmp_path)  # empty dir
    report = v.validate_xml(b"<rsm:CrossIndustryInvoice/>")
    # We expect the syntax error or warnings about unavailable artifacts, never a crash.
    assert isinstance(report.to_dict(), dict)


def test_validator_reports_syntax_error(tmp_path: Path) -> None:
    v = FacturXValidator(specs_root=tmp_path)
    report = v.validate_xml(b"<not-xml")
    assert report.valid is False
    assert any(f.code == "XML_PARSE" for f in report.findings)


def test_extractor_returns_none_when_no_embedded_file() -> None:
    ext = PDFExtractor()
    assert ext.extract(b"%PDF-1.7\n%nope") is None
