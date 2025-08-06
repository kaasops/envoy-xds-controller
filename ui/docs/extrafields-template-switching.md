# ExtraFields Template Switching Fix

## Issue Description

In the virtual service creation and editing form, when a user selects one template with extraFields, fills in the dynamic fields, and then selects another template, the extraFields from the previous template remain in memory. This should not happen - when switching templates, only the extraFields defined in the new template should be preserved.

## Root Cause

The issue was in the `extraFieldsTabVs.tsx` component, specifically in the `useEffect` that handles template changes. The previous implementation was merging the default values with the current values:

```typescript
// Merge with existing values to preserve them
if (Object.keys(initialExtraFields).length > 0) {
  setValue('extraFields', { ...currentValues, ...initialExtraFields })
}
```

This preserved all existing extraFields, even those that weren't defined in the new template.

## Solution

The solution was to modify the `useEffect` to:

1. Create a new empty extraFields object instead of merging with existing values
2. Only include fields that are defined in the new template
3. Preserve values for fields that exist in both the old and new templates
4. Set default values for fields that don't have a value
5. Clear all extraFields if the template has no extraFields

```typescript
useEffect(() => {
  // Create a new extraFields object
  const newExtraFields: Record<string, string> = {}
  
  if (extraFields.length > 0) {
    // Create a set of valid field names from the current template
    const validFieldNames = new Set(extraFields.map(field => field.name))
    
    // Get current values
    const currentValues = currentExtraFields || {}
    
    // For each field in the template
    extraFields.forEach(field => {
      // If the field already has a value in the current form, preserve it
      if (currentValues[field.name]) {
        newExtraFields[field.name] = currentValues[field.name]
      } 
      // Otherwise, use the default value if available
      else if (field.default) {
        newExtraFields[field.name] = field.default
      }
      // If no default, initialize with empty string
      else {
        newExtraFields[field.name] = ''
      }
    })
    
    // Set the new extraFields, completely replacing the old ones
    // This ensures fields from previous templates are not preserved
    setValue('extraFields', newExtraFields)
  } else {
    // If there are no extraFields in the template, clear all extraFields
    setValue('extraFields', {})
  }
}, [selectedTemplateUid, extraFields, setValue])
```

## Testing

To verify that the fix works correctly:

1. Create a new virtual service
2. Select a template with extraFields (e.g., microservice-gateway-template)
3. Fill in some of the extraFields
4. Switch to a different template (e.g., auth-service-template)
5. Verify that only the extraFields defined in the new template are displayed
6. Verify that extraFields from the previous template that aren't in the new template are not preserved
7. Switch back to the original template and verify that the extraFields are reset to their default values

## Additional Notes

This fix ensures that the UI properly clears extraFields when switching templates, which aligns with the server-side validation that was previously implemented. The server already validates that only extraFields defined in the template are present in the VirtualService, but now the UI also prevents invalid extraFields from being submitted.