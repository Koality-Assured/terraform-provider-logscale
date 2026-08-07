# Provider Research Reference

This directory is the in-repo research wiki for the LogScale Terraform provider.

It is intended to:

- collect researched API behavior in a durable place
- summarize what has been validated from manual GraphQL probes and official docs
- preserve bug-fix and runtime lessons so future work does not rediscover them
- track non-code dependencies and external references that resources may rely on
- support future implementation without forcing each work session to rediscover the same information

This directory is documentation-only and should remain outside of any source/build path.

## Major Categories

- [Provider Expansion](./provider-expansion/README.md)

## Usage Rules

- organize research by major category and then by resource family
- keep category `README.md` files as summary/index pages
- place deeper notes in separate topic files under the category
- prefer official documentation and local tenant-validation notes over memory
- record both confirmed capabilities and unresolved questions
- note when a resource depends on external identities or assets that may not be managed by this provider
