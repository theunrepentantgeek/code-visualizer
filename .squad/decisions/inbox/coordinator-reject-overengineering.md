### 2026-05-06T20:52:00Z: Reject over-complex refactoring designs

**By:** Bevan Arps (reviewer)
**What:** Rejected Ripley's designs for #158 (renderer unification) and #152 (command workflow extraction). The proposed patterns (Strategy/Visitor, Template Method/pipeline builder) introduce too much complexity to simple problems and would actively obfuscate the code.
**Why:** Simplicity over abstraction — don't add indirection that makes code harder to follow, even if it reduces line count.
