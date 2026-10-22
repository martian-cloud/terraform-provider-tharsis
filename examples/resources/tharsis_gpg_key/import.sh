# A GPG key can be imported by its path, which is the group path plus the key
# fingerprint.
terraform import tharsis_gpg_key.this my-group/C48F7F19B14E59433DF9402341F57AB6741F92F1

# The key's TRN also works.
terraform import tharsis_gpg_key.this trn:gpg_key:my-group/C48F7F19B14E59433DF9402341F57AB6741F92F1

# So does its ID, shown by the "id" attribute or in the Tharsis UI.
terraform import tharsis_gpg_key.this <gpg-key-id>
