# Finch Brand Assets

This directory contains official branding assets for Finch.

## Files

| File | Description |
|------|-------------|
| `finch-logo-horizontal.svg` | Full horizontal logo — bird mark + wordmark, light mode |
| `finch-logo-horizontal-dark.svg` | Full horizontal logo — bird mark + wordmark, dark mode |
| `finch-icon-circle.svg` | Bird mark inside a circle, light mode |
| `finch-icon-circle-dark.svg` | Bird mark inside a circle, dark mode |

## Colours

| Role | Hex |
|------|-----|
| Brand navy | `#20415a` |
| White | `#ffffff` |

## Usage

- Use the **light** variants on white or light-coloured backgrounds.
- Use the **dark** variants on dark backgrounds (e.g. GitHub dark theme).
- For GitHub `README.md` use a `<picture>` element to switch automatically:

```html
<picture>
  <source media="(prefers-color-scheme: dark)" srcset=".brand/finch-logo-horizontal-dark.svg">
  <img src=".brand/finch-logo-horizontal.svg" alt="Finch - The Minimal Observability Infrastructure" width="300">
</picture>
```

## Notes

- All assets have a transparent background unless otherwise noted
  (`finch-icon-circle.svg` has a white fill inside the circle stroke).
- Do not alter the wordmark paths or the proportions of the bird mark.
