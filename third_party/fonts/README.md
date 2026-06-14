# Third-party fonts

The application self-hosts these typefaces. The `.woff2` files are embedded in
the binary and served from `/static/fonts/` — no CDN or Google Fonts is ever
contacted (a font CDN would see every visitor's IP).

| Family         | Weights | Source binaries                                  | Licence                                   |
| -------------- | ------- | ------------------------------------------------ | ----------------------------------------- |
| Inter Tight    | 400, 500 | `internal/web/assets/fonts/inter-tight-*.woff2`    | SIL OFL 1.1 — [`Inter-Tight-OFL.txt`](Inter-Tight-OFL.txt) |
| JetBrains Mono | 400, 500 | `internal/web/assets/fonts/jetbrains-mono-*.woff2` | SIL OFL 1.1 — [`JetBrains-Mono-OFL.txt`](JetBrains-Mono-OFL.txt) |

The licence files in this directory are the verbatim upstream texts:

- Inter — https://github.com/rsms/inter (`LICENSE.txt`)
- JetBrains Mono — https://github.com/JetBrains/JetBrainsMono (`OFL.txt`)

The application code itself is licensed separately under AGPL-3.0; see the
top-level [`LICENSE`](../../LICENSE).
