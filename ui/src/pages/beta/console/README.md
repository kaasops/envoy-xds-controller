# Beta console sections

The section registry in [sections.ts](sections.ts) drives the routes, sidebar and section search. A section is a lazily loaded React component. The node snapshot and comparison routes are registered separately in App.tsx so existing direct links continue to work.

## Add a section

1. Add a default-exported component in `sections/<slug>.tsx` and register its slug, menu group, title and lazy import in `sections.ts`. The route will be `/beta/<slug>`.
2. Read the data through the existing authenticated REST or Connect client. Scope the query key to its inputs and pass the cancellation signal to the client. Use `manualQueryOptions` from `../api.ts` and expose an explicit refresh. Loading a list must not trigger snapshot builds.
3. Render the rows with [SectionTable](SectionTable.tsx). Give each column a stable key, a label and a value. Provide a stable row ID and a detail renderer to enable the split view. Connect query progress, errors and timestamps to `loading`, `error` and `updatedAt`. For a browser-only file import, use local filter state.
4. Open the section directly and through the menu. Check an empty response, a failed refresh, keyboard navigation, a narrow viewport and a direct link with a filter. Use `?q=term` for text, `?by=columnKey&value=exactValue` for a property and `?selected=rowId` for the detail panel. Encode values with `URLSearchParams`.
