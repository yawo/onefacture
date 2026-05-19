# RESEARCH.md — Phase 0: Research & Specifications

## 1. Regulatory Context (French E-Invoicing 2026)
The reform mandates the use of accredited platforms (PDP) or the public portal (PPF) using a "Y" schema.

### Key AFNOR Standards
- **XP Z12-012 (Data & Formats):** Defines the semantic data model (EN 16931) and syntax (UBL 2.1, CII D22B, Factur-X 1.08).
- **XP Z12-013 (API Connectivity):** Standardizes RESTful APIs for communication between ERPs/SaaS and PDPs.
- **XP Z12-014 (Business Use Cases):** Detailed scenarios (Credit notes, Subcontracting, etc.).

## 2. Validation Artifacts (Factur-X 1.08 / ZUGFeRD 2.4)
The standard is technically identical between France and Germany as of late 2025/early 2026.

### XSD Schemas (Syntax)
- **Source:** [FNFE-MPE Official](https://fnfe-mpe.org/factur-x/)
- **Mirror:** [akretion/factur-x/facturx/xsd](https://github.com/akretion/factur-x/tree/master/facturx/xsd)
- **Base Syntax:** UN/CEFACT CII D22B.

### Schematrons (Business Rules)
- **Ruleset:** EN 16931-3-2 (CII).
- **French Specifics:** AFNOR XP Z12-012 adds specific rules for SIREN/SIRET and French tax particularities.
- **Tools:** Official validation usually requires an XSLT engine (Saxon/LXML).

## 3. Official DGFiP / PPF Technical Specifications
The "Dossier des Spécifications Externes" (DSE) is the bible for the PPF integration.

- **Current Version:** v3.x (as of 2026).
- **OpenAPI / Swagger:** Includes APIs for:
    - **Directory (Annuaire):** SIREN -> Routing information.
    - **Submission (Flux):** Sending invoices in Factur-X/UBL/CII.
    - **Lifecycle (Statuts):** CDAR (Comptes-rendus d'Activité).
    - **E-reporting:** Transaction & Payment data.
- **Official Portal:** [impots.gouv.fr/specifications-externes-b2b](https://www.impots.gouv.fr/specifications-externes-b2b)
- **Developer Portal:** [AIFE Developer Portal (PISTE)](https://developer.aife.economie.gouv.fr/)

## 4. Go Implementation Strategy

### Libraries
- **Logic & Generation:** `github.com/dotwavehq/go-einvoice` (EN 16931 compliant).
- **PDF/A-3 & Embedding:** `github.com/angelodlfrtr/go-invoice-generator`.
- **Validation:** Go-native for XSD, but Python Sidecar (lxml) or Java (Mustang) for Schematron.

### Validation Pipeline (6 Layers)
1. **Container Check:** Validate PDF/A-3 compliance.
2. **XML Extraction:** Extract `factur-x.xml` from PDF.
3. **XSD Validation:** Check XML against UN/CEFACT CII schemas.
4. **Schematron (EN 16931):** Generic European business rules.
5. **Schematron (AFNOR):** French-specific rules (XP Z12-012).
6. **Custom Business Rules:** Internal logic (e.g., duplicate detection, org-specific constraints).

## 5. Directory Structure for Artifacts
```
/docs/specs/
  ├── afnor/        # XP Z12-012/013/014 summaries
  ├── dgfip/        # Swagger/OpenAPI definitions
  └── validation/
      ├── xsd/      # Official XSDs (CII D22B)
      └── schematron/ # Official .sch or .xsl files
```
