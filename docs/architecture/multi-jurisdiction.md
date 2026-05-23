# Framework multi-juridiction

Le module `internal/jurisdiction` isole les profils pays/formats du coeur API.

## Profils initiaux

- `FR`: Factur-X EN16931, formats Factur-X, CII, UBL.
- `EU`: PEPPOL BIS Billing, format UBL.

## Extension

Ajouter un pays ou un profil ne doit pas modifier les endpoints REST principaux:

1. Enregistrer un `jurisdiction.Profile` dans `NewRegistry()`.
2. Brancher les règles de validation/génération propres au pays (si nécessaire).
3. Ajouter les tests de mapping format/profil.

Exemple d'ajout pour ViDA (2028+) :

```go
r.Register(Profile{
    CountryCode: "EU-ViDA",
    Name:        "ViDA / EN16931 (2028+)",
    Formats:     []string{"CII", "UBL"},
})
```

## Nouveaux adaptateurs PA

Les adaptateurs (Cegid, Qonto, etc.) suivent le même pattern que Pennylane/Docaposte :

- Implémentent `PAAdapter`
- Utilisent `sandbox.Client` pour le développement
- Sont enregistrés dans `internal/adapters/registry.NewDefault()`

Cela permet d'ajouter un nouvel adaptateur sans toucher au cœur de l'API ni au routing.

Cette structure prépare l'ajout de nouveaux PDP et l'évolution vers PEPPOL/ViDA sans dette technique.
