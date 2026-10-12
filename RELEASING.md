# Releasing terraform-provider-logscale

Releases are fully automated via GitHub Actions (`.github/workflows/release.yml`) when a new `v*` tag is pushed to the repository.

## First-Time Setup (One-time only)

To publish this provider to the Terraform Registry, you need a GPG key to sign the releases.

1. **Generate a GPG Key**:
   ```sh
   gpg --batch --passphrase '' --quick-gen-key 'Koality Assured <maintainers@koality-assured.com>' rsa4096 default never
   ```

2. **Export the keys**:
   ```sh
   # Export Public Key (Upload this to Terraform Registry)
   gpg --armor --export 'Koality Assured' > public.asc

   # Export Private Key
   gpg --armor --export-secret-keys 'Koality Assured' > private.asc
   ```

3. **Configure GitHub Repository Secrets**:
   Go to the repository settings -> Secrets and variables -> Actions, and add:
   - `GPG_PRIVATE_KEY`: The contents of `private.asc`
   - `PASSPHRASE`: The passphrase used when generating the key (if any)

4. **Upload Public Key to Terraform Registry**:
   Upload `public.asc` to the provider settings in registry.terraform.io.

## Creating a Release

To trigger a release:

1. Update the `CHANGELOG.md` (if applicable) and merge into `master`.
2. Create and push a new tag matching the version you want to release:
   ```sh
   git tag v1.0.0
   git push origin v1.0.0
   ```
3. The GitHub Actions workflow will automatically build, sign, and create a GitHub Release.
4. The Terraform Registry will automatically detect the new release via its webhook or polling mechanism.
