# Data Model: Use Goldie for Golden File Testing

**Date**: 2026-04-05  
**Feature**: 003-use-goldie

## Overview

This feature introduces no new entities or data model changes. It replaces handwritten golden file infrastructure with the Goldie library, which is a test-only dependency.

## Entities

No new entities. The existing golden file PNG images in `internal/render/testdata/` remain unchanged in format and content. Goldie treats them as opaque `[]byte` data for comparison purposes.

## State Transitions

N/A — no state is introduced or modified.

## Validation Rules

N/A — Goldie performs byte-level equality comparison on golden files. No custom validation logic is needed.
