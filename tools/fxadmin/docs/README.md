<!--
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
-->

# fxadmin — Fabric-X Reconfiguration Admin CLI

`fxadmin` is a command-line tool for **administering the configuration of
Fabric-X**. It lets administrators pull the
current configuration from the running network, edit it, collect the required
admin endorsements, wrap the result into a submittable transaction, and
broadcast that transaction to the ordering service.

---

## The reconfiguration flow
A configuration change is produced by the following steps:

**Manual mode**
1. Get the last config block from the orderer.
2. Decode the block to json.
3. Edit the json manually.
4. Compute the config update from the current config json and the modified json.
5. Sign a config-update as admin endorser, typically more than one endorser.
6. Merge admin signatures.
7. Prepare the TX proposal for submission by wrapping it in an envelope and signing it with a Channel Writer principal (can be one of the admins).
8. Submit the config-update to all routers.
9. Follow the assembler ledger until the new config is committed in all the parties.

NOTE: Steps 2–4 describe the manual process for building a configuration change.
For the most common changes - such as endpoints, node certificates, CA lists, 
and adding or removing organizations and parties - use the [Modify Commands](#modify-commands-automation-of-the-manual-step) instead. 
These commands edit the config block directly and allow multiple changes to be accumulated into a single update.
The manual process remains fully supported; the modify commands simply automate the common cases.

**Auto mode**
1. Get the last config block from the orderer.
2. Copy the last config block to "next" block.
3. Apply one or more modifications to next block.
4. Compute the config update from the current config block and the next block.
5. Sign a config-update as admin endorser, typically more than one endorser.
6. Merge admin signatures.
7. Prepare the TX proposal for submission by wrapping it in an envelope and signing it with a Channel Writer principal (can be one of the admins).
8. Submit the config-update to all routers.
9. Follow the assembler ledger until the new config is committed in all the parties.

---

## Prerequisites

- An **admin identity** for at least one organization (MSP directory).
- The admin's TLS client certificate/key and whether TLS is enabled.
- The orderer TLS root CAs and the orderer endpoints for the assemblers and routers (retrieved from the **current config block** of the channel).    
The first time you run `fxadmin`, use the channel **genesis block**. After each committed reconfiguration, use the newly produced config block.



## Build

`fxadmin` is built from this repository:

```bash
go build -o bin/fxadmin ./tools/fxadmin
```

Verify the build:

```bash
./bin/fxadmin --help
```

---

## Configuration file (`admin.yaml`)

The file describes the admin's **MSP identity**, and its
**TLS** certificate and key.

**Example**
```yaml
# admin configuration for orderer org1

msp:
  localMspID: org1
  localMspDir: ARTIFACTS_DIR/ordererOrganizations/org1/users/Admin@org1/msp

tls:
  enabled: true
  clientCert: ARTIFACTS_DIR/ordererOrganizations/org1/users/Admin@org1/tls/client.crt
  clientKey:  ARTIFACTS_DIR/ordererOrganizations/org1/users/Admin@org1/tls/client.key
```

**Fields**

| Section   | Field          | Description                                                                                                                              |
| --------- | -------------- |------------------------------------------------------------------------------------------------------------------------------------------|
| `msp`     | `localMspID`   | MSP ID of the admin's organization (e.g. `org1`). Used to build the signer's identity.                                                   |
| `msp`     | `localMspDir`  | Path to the admin's MSP directory (signing key + certificates).                                                                          |
| `tls`     | `enabled`      | Enable TLS. `false` (the default) = no TLS; `true` without `clientCert`/`clientKey` = server-side TLS (the server is verified using the orderer root CAs taken from the current config block); `true` with `clientCert`+`clientKey` = mutual TLS. |
| `tls`     | `clientCert`   | Admin's TLS **client certificate** (required for mutual TLS).                                                                            |
| `tls`     | `clientKey`    | Admin's TLS **client private key** (required for mutual TLS).                                                                            |

> **Note** — `clientCert` and `clientKey` must be provided together. If only one
> is set, mutual TLS is disabled and a warning is logged.

---

## Commands

### General flags

The following flags apply to all commands:

| Flag           | Description                       |
| -------------- | --------------------------------- |
| `-h`, `--help` | Show help for the command.        |
| `--version`    | Print the `fxadmin` version.      |

---

### Ledger Commands

```bash
fxadmin ledger [ledger flags] <command> [command flags]
```
#### Flags

These flags are common to all ledger commands:

| Flag              | Required | Description                                              |
| ----------------- | :------: |----------------------------------------------------------|
| `--current-block` |   yes    | Current config block containing the assembler's endpoints |
| `--config`        |   yes    | Admin configuration file (identity, TLS)                 |

#### Commands
```bash
# Ask for the ledger height
fxadmin ledger height

# Pull the latest block
fxadmin ledger block latest --output block.pb

# Pull block number 123
fxadmin ledger block 123 --output block.pb

# Pull the latest ledger configuration
fxadmin ledger config latest --output config.pb
```

---

#### Get the ledger height
Print the **height** of the ledger.

```bash
fxadmin ledger \
  --config admin.yaml \
  --current-block current_block.pb \
  height
```
---

#### Get the latest block
Fetch the latest committed block from an assembler and write it to a file.

```bash
fxadmin ledger \
  --config admin.yaml \
  --current-block current_block.pb \
  block latest \
  --output latest_block.pb
```
---

#### Get a specific block
Fetch a specific block by number and write it to a file.

```bash
fxadmin ledger \
  --config admin.yaml \
  --current-block current_block.pb \
  block 123 \
  --output block_123.pb
```
---

#### Get the last config block
Fetch the latest committed **config** block from an assembler and write it to a file.

```bash
fxadmin ledger \
  --config admin.yaml \
  --current-block current_block.pb \
  config latest \
  --output last_config.pb
```
---

> **Planned (not yet implemented)** — support environment variables as defaults
> for the common flags, to reduce repetitive typing and ease automation:
>
> ```bash
> export FXADMIN_CONFIG=admin.yaml
> export FXADMIN_CURRENT_BLOCK=current_block.pb
> ```

---
### Decode Command

The decode command extracts the `common.Config` embedded in a binary config block
(.pb) and writes it as a human-readable JSON file for editing.
This command is typically used after retrieving the latest configuration from the ledger.

```bash
fxadmin decode <config_block.pb> --output <current_config.json>
```

Example:
```bash
fxadmin decode last_config.pb --output current_config.json
```

**Arguments**

| Argument            | Required | Description                                      |
|---------------------| :------: |--------------------------------------------------|
| `<config_block.pb>` |   yes    | Path to the protobuf config block file to decode |

**Flags**

| Flag       | Required | Description                                      |
|------------| :------: |--------------------------------------------------|
| `--output` |   yes    | Path to the output `common.Config` JSON file, derived from the block |

---
### Manual step
After decoding, copy `current_config.json` to `modified_config.json` and edit `modified_config.json`
manually to describe the desired configuration (for example: rotate a
certificate, add or remove an organization/party, change an endpoint, or adjust
a batching/consensus parameter). Leave `current_config.json` untouched.

---

### Modify Commands (Automation of the Manual step)
The `modify` commands automate the **manual** change (decode → hand-edit) for the **most popular** configuration changes. 
They edit a config **block file** directly. Several changes can accumulate into the same block and
commit as a single config update. The manual step remains available for any change the modify commands do not cover.

**The flow**

```bash
# 1. Get the latest config block.
fxadmin ledger --config admin.yaml --current-block current_block.pb config latest --output current.pb

# 2. Copy it to a "next" block.
cp current.pb next.pb

# 3. Apply one or more modify commands to next.pb (changes accumulate).
# For example: change the port of the batcher of party1, shard1 
fxadmin modify party node --party 1 --role batcher --shard 1 --port 9014 --block next.pb
#   … more modify commands may be run against the same next.pb …

# 4. Compute the config update from the two blocks (block mode).
fxadmin compute-update --pb current.pb next.pb --output config_update.pb

# 5. Endorse → merge → send → follow  (unchanged - see the Transaction Command).
```

`modify` edits the block on disk and never contacts the network. 
Keep `current.pb` untouched for the `compute-update --pb` step in #4.

**When a command is rejected**

A modify command first checks its arguments, including the certificate files it reads. It then
checks the edited config with the rules of the channel config, and accepts or rejects the edited
config exactly as the channel config does. A rejected command prints the reason and leaves the
block file unchanged. The changes that earlier modify
commands made to the same block stay. Fix the cause and run the command again. The rules that the
orderer adds on top of the channel config are checked when the config update is submitted. Each
command lists its rejections under **When the command fails**.

**Application orgs and orderer orgs are different orgs.** fxadmin assumes that each org is either
an application org (in the `Application` group) or an orderer org (in the `Orderer` group), with
its own MSP. An org that is in both groups is not supported. A change to its MSP is rejected,
because the channel config requires both copies of the MSP to be identical
(`Attempted to define two different versions of MSP`).

**Commands**

```bash
# Application
fxadmin modify app add                --org=<org.yaml>            --block <next.pb>
fxadmin modify app remove             <org>                       --block <next.pb>
fxadmin modify app known-certs  add|remove|set  --org <org> --cert <pem>…   --block <next.pb>

# Orderer organizations
fxadmin modify orderer known-certs  add|remove|set  --org <org> --cert <pem>…   --block <next.pb>

# Orderer (ARMA parties)
fxadmin modify party add              --party=<party.yaml>        --block <next.pb>
fxadmin modify party remove           <party-id>                  --block <next.pb>

# Party's node (any subset of its fields)
fxadmin modify party node   --party <N> --role <role> [--shard <S>] \
    [--host <H>] [--port <P>] [--tls-cert <pem>] [--sign-cert <pem>]  --block <next.pb>

# A party's CA / TLS-CA certificate lists
fxadmin modify party ca  add|remove|set  --party <N> \
    [--sign-cert <pem>…] [--tls-cert <pem>…]                         --block <next.pb>

# The CA the node OUs of a party's org MSP pin
fxadmin modify party ca  node-ous  --party <N> --sign-cert <pem>     --block <next.pb>
```

**Common flags on every command**

| Flag | Required | Description                                               |
|------|:--------:|-----------------------------------------------------------|
| `--block` | yes | Config block file to edit **in place** (the "next" block) |

Certificate flags always take a PEM **file path**, never inline certificate
bytes; the tool reads and parses the file.

For modify party node and modify party ca, omitted flags leave the corresponding fields unchanged, 
while supplied flags update them. Multiple flags can be provided in a single command to update several fields at once.

---

**modify app add**

Add an application organization to the `Application` group.

```bash
fxadmin modify app add --org=peer2.yaml --block next.pb
```

`peer2.yaml` is the organization definition. It has two keys:

- `name`: the org's key under `Application`.
- `config_group`: the config group of the organization
  (`channel_group.groups.Application.groups.<name>`), in the form it has in the config block.

The command copies the config group into the block as it is, and fills in nothing.

**Certificates.** Give every certificate and CRL as the path of its PEM file, absolute or
relative. A relative path is relative to the directory of `peer2.yaml`. Each file must hold
exactly one PEM block, an X.509 certificate, or a CRL in `revocation_list`. A file of several
blocks, such as a certificate followed by its private key or a bundle of certificates, is
rejected: give each certificate in a file of its own.

- `root_certs`, `intermediate_certs`, `tls_root_certs` and `tls_intermediate_certs` must hold CA
  certificates.
- In each entry of `organizational_unit_identifiers`, the `certificate` field must be a CA
  certificate. The entry's `organizational_unit_identifier` is the OU name, a plain string.
- In each node OU of `fabric_node_ous` (`client_ou_identifier`, `peer_ou_identifier`,
  `admin_ou_identifier`, `orderer_ou_identifier`), the `certificate` field must be a CA
  certificate, or empty (`""`) to accept an identity issued by any of the MSP's CAs. This holds
  whether node OUs are enabled or not.
- `admins` and `known_certs` must hold identity certificates, not CA certificates.
- A list can't hold the same certificate twice.

```yaml
# peer2.yaml
name: peer2                          # the org's key under Application

config_group:                        # → Application/groups/<name>
  version: "0"
  mod_policy: Admins
  groups: {}

  policies:                          # Readers, Writers, Admins, Endorsement
    Readers:
      version: "0"
      mod_policy: Admins
      policy:
        type: 1                      # 1 = SIGNATURE
        value:                       # OR('peer2.member')
          version: 0
          identities:
            - principal_classification: ROLE
              principal:
                msp_identifier: peer2
                role: MEMBER
          rule:
            n_out_of:
              n: 1
              rules:
                - signed_by: 0

    Writers:
      version: "0"
      mod_policy: Admins
      policy:
        type: 1
        value:                       # OR('peer2.member')
          version: 0
          identities:
            - principal_classification: ROLE
              principal:
                msp_identifier: peer2
                role: MEMBER
          rule:
            n_out_of:
              n: 1
              rules:
                - signed_by: 0

    Admins:
      version: "0"
      mod_policy: Admins
      policy:
        type: 1
        value:                       # OR('peer2.admin')
          version: 0
          identities:
            - principal_classification: ROLE
              principal:
                msp_identifier: peer2
                role: ADMIN
          rule:
            n_out_of:
              n: 1
              rules:
                - signed_by: 0

    Endorsement:
      version: "0"
      mod_policy: Admins
      policy:
        type: 1
        value:                       # OR('peer2.member')
          version: 0
          identities:
            - principal_classification: ROLE
              principal:
                msp_identifier: peer2
                role: MEMBER
          rule:
            n_out_of:
              n: 1
              rules:
                - signed_by: 0

  values:
    MSP:
      version: "0"
      mod_policy: Admins
      value:
        type: 0                      # 0 = FABRIC
        config:                      # FabricMSPConfig
          name: peer2                # MSP ID; the msp_identifier of the policies
          root_certs:
            - crypto/peer2/ca/ca-cert.pem
          intermediate_certs: []
          admins:
            - crypto/peer2/users/Admin/msp/signcerts/cert.pem
          revocation_list: []
          signing_identity: null
          organizational_unit_identifiers: []
          crypto_config:
            signature_hash_family: SHA2
            identity_identifier_hash_function: SHA256
          tls_root_certs:
            - crypto/peer2/tlsca/tlsca-cert.pem
          tls_intermediate_certs: []
          fabric_node_ous: null
          known_certs:
            - crypto/peer2/users/User1/msp/signcerts/cert.pem
            - crypto/peer2/users/User2/msp/signcerts/cert.pem
```

**Flags**

| Flag | Required | Description                               |
|------|:--------:|-------------------------------------------|
| `--org` | yes | Path to the organization definition YAML  |
| `--block` | yes | Config block to edit in place             |

**When the command fails**

The command leaves the block unchanged if:

- `peer2.yaml` cannot be parsed, has no `name` or no `config_group`, or has a key other than these two.
- The `name` is not a string, or the `config_group` is not a map.
- A key of `config_group` is not a field of the block structure.
- A certificate or CRL file can't be read, or doesn't hold what its field needs (see
  **Certificates**).
- The block has no `Application` group.
- An application org with the same `name` is already in the block.
- The edited config is not a valid channel config. This also catches a `config_group` with no MSP,
  and an MSP whose MSP ID is already the MSP ID of an org in the block but whose config differs
  from that org's MSP.

---
**modify app remove**

Remove an application organization.

```bash
fxadmin modify app remove peer2 --block next.pb
```

The command removes the org's whole config group (`channel_group.groups.Application.groups.<org>`):
its MSP, its policies and every other value it holds. Nothing else in the block changes.

**What the admins are responsible for.** A config update is accepted when the admins that its
policies require sign it, and the new config is valid. Nothing checks what the rest of the config
still refers to the removed org by: that is the responsibility of the admins who sign
the update. The command helps by warning about these references, and it rejects the one removal that
would open the channel to anyone (see **When the command fails**).

**References to the org.** The command keeps what the rest of the config still refers to the org by,
and warns about it, naming each reference:

- an ACL of the `Application` group that refers to one of the org's policies, such as
  `/Channel/Application/<org>/Readers`. The ACL then denies every request;
- a signature policy that names the org's MSP ID, when no other org has that MSP ID. The org can then
  no longer satisfy the policy, for example `AND('org1.admin', '<org>.admin')`.

Update them in a later config update. Each config update is checked against the policies of the
config before it, so the later update works as long as the policy that guards the change, usually
`Admins`, does not need the removed org. A policy that guards changes and needs the org must be
changed before the org is removed, in a config update of its own: once the org is gone, that policy
can no longer be satisfied, and nothing it guards can be changed again.

**Arguments**

| Argument | Required | Description                                       |
|----------| :------: |---------------------------------------------------|
| `<org>` | yes | The org's name under `Application`, not its MSP ID |

**When the command fails**

The command leaves the block unchanged if:

- The block has no `Application` group, or no org named `<org>` in it. The error lists the orgs of
  the group, or names the org whose MSP ID `<org>` is.
- `<org>` is the only application org. With no org, the policies of the `Application` group would
  be satisfied by anyone, even without a signature.
- The edited config is not a valid channel config.

---

**modify app | orderer known-certs add | remove | set**

Change the `known_certs` list of an org's MSP: of an application org with `app`, or of an orderer
org with `orderer`. A client can sign with the ID of a known certificate instead of sending the
certificate. The MSP resolves that ID only through its known certificates.

```bash
fxadmin modify app known-certs add        --org peer1 --cert known1.pem --block next.pb
fxadmin modify app known-certs remove     --org peer1 --cert known1.pem --block next.pb
fxadmin modify orderer known-certs set    --org org1  --cert admin.pem --cert client.pem --block next.pb
```

**Subcommands**

| Subcommand | Effect on the org's `known_certs` |
|------------|-----------------------------------|
| `add`      | Append the given cert(s). A cert that is already in the list is skipped with a message. |
| `remove`   | Drop the given cert(s). Fails if a cert is not in the list. The list may be left empty. |
| `set`      | Replace the whole list with exactly the given cert(s), in the order given. |

**Flags**

| Flag | Required | Description |
|------|:--------:|-------------|
| `--org` | yes | The org's name in its group (`Application` for `app`, `Orderer` for `orderer`), not its MSP ID |
| `--cert` | yes (≥1) | PEM path of a known certificate (repeatable) |
| `--block` | yes | Config block to edit in place |

Each file must hold exactly one PEM-encoded X.509 certificate of an identity, not of a CA. The MSP
never accepts a CA certificate as an identity. The same certificate can't be given twice. A file of
several certificates, such as a bundle of known certificates, is rejected: give each certificate in
a file of its own.

The org's MSP must be a Fabric (X.509) MSP. An org with another MSP type, such as idemix, is
rejected.

Application orgs and orderer orgs are assumed to be different orgs (see
[Modify Commands](#modify-commands-automation-of-the-manual-step)): only the MSP of the named org,
in its own group, changes.

**Removing a known certificate** breaks the clients that still sign with its certificate ID. Once
the update commits, their transactions are rejected. It is the admin's responsibility to remove
only certificates that are no longer in use.

**When the command fails**

The command leaves the block unchanged if:

- **Every subcommand:**
  - a certificate file can't be read, isn't a PEM-encoded X.509 certificate, holds more than one
    PEM block, is a CA certificate, or is given twice;
  - the group has no org with that name;
  - the org's MSP is not a Fabric (X.509) MSP;
  - the edited config is not a valid channel config. For example, the org's MSP rejects a known
    certificate that none of its CAs issued.
- **`remove`:** a certificate is not in the list.

---

**modify party add**

Add a new ARMA party together with its new orderer organization.

```bash
fxadmin modify party add --party=party5.yaml --block next.pb
```

`party5.yaml` is the party definition. It has two sections, each mirroring its form in the
config block:

- `Party`: the party's entry in the ARMA shared config
  (`channel_group.groups.Orderer.values.ConsensusType.value.metadata.PartiesConfig`).
- `Org`: the config group of the party's orderer organization
  (`channel_group.groups.Orderer.groups.<org>`), plus a `Name` key with the org's key under
  `Orderer`.

The command copies both sections into the block as they are, and fills in nothing.

**Certificates.** Give every certificate and CRL as the path of its PEM file, absolute or
relative. A relative path is relative to the directory of `party5.yaml`. Each file must hold
exactly one PEM block, an X.509 certificate, or a CRL in `revocation_list`. A file of several
blocks, such as a certificate followed by its private key or a bundle of certificates, is
rejected: give each certificate in a file of its own.

- `CACerts` and `TLSCACerts` of `Party`, and `root_certs`, `intermediate_certs`, `tls_root_certs`
  and `tls_intermediate_certs` of the org's MSP must hold CA certificates.
- In each entry of `organizational_unit_identifiers`, the `certificate` field must be a CA
  certificate. The entry's `organizational_unit_identifier` is the OU name, a plain string.
- In each node OU of `fabric_node_ous` (`client_ou_identifier`, `peer_ou_identifier`,
  `admin_ou_identifier`, `orderer_ou_identifier`), the `certificate` field must be a CA
  certificate, or empty (`""`) to accept an identity issued by any of the MSP's CAs. This holds
  whether node OUs are enabled or not.
- `admins` and `known_certs` must hold identity certificates, not CA certificates.
- A list can't hold the same certificate twice.

```yaml
# party5.yaml
Party:                               # → Orderer ConsensusType metadata: PartiesConfig[]
  PartyID: 5                         # must be MaxPartyID + 1 of the block
  CACerts:                           # the org's CA certificates
    - crypto/org5/ca/ca-cert.pem
  TLSCACerts:                        # the org's TLS CA certificates
    - crypto/org5/tlsca/tlsca-cert.pem
  RouterConfig:
    node_config:
      host: router.org5.com
      port: 7050
      client_tls_cert: crypto/org5/router/tls/server.crt   # may be the same file as server_tls_cert
      server_tls_cert: crypto/org5/router/tls/server.crt
      sign_cert: crypto/org5/router/msp/signcerts/cert.pem
  BatchersConfig:                    # one per shard; shard IDs start at 1
    - shard_id: 1
      node_config:
        host: batcher1.org5.com
        port: 7070
        client_tls_cert: crypto/org5/batcher1/tls/server.crt
        server_tls_cert: crypto/org5/batcher1/tls/server.crt
        sign_cert: crypto/org5/batcher1/msp/signcerts/cert.pem
  ConsenterConfig:
    node_config:
      host: consenter.org5.com
      port: 7052
      client_tls_cert: crypto/org5/consenter/tls/server.crt
      server_tls_cert: crypto/org5/consenter/tls/server.crt
      sign_cert: crypto/org5/consenter/msp/signcerts/cert.pem
  AssemblerConfig:
    node_config:
      host: assembler.org5.com
      port: 7060
      client_tls_cert: crypto/org5/assembler/tls/server.crt
      server_tls_cert: crypto/org5/assembler/tls/server.crt
      sign_cert: crypto/org5/assembler/msp/signcerts/cert.pem
Org:                                 # → Orderer/groups/<Name>
  Name: org5                         # the org's key under Orderer
  version: "0"
  mod_policy: Admins
  groups: {}
  policies:                          # Readers, Writers, Admins, Endorsement
    Readers:
      version: "0"
      mod_policy: Admins
      policy:
        type: 1                      # 1 = SIGNATURE
        value:                       # OR('org5.member')
          version: 0
          identities:
            - principal_classification: ROLE
              principal: { msp_identifier: org5, role: MEMBER }
          rule: { n_out_of: { "n": 1, rules: [ { signed_by: 0 } ] } }
    Writers:
      version: "0"
      mod_policy: Admins
      policy:
        type: 1
        value:                       # OR('org5.member')
          version: 0
          identities:
            - principal_classification: ROLE
              principal: { msp_identifier: org5, role: MEMBER }
          rule: { n_out_of: { "n": 1, rules: [ { signed_by: 0 } ] } }
    Admins:
      version: "0"
      mod_policy: Admins
      policy:
        type: 1
        value:                       # OR('org5.admin')
          version: 0
          identities:
            - principal_classification: ROLE
              principal: { msp_identifier: org5, role: ADMIN }
          rule: { n_out_of: { "n": 1, rules: [ { signed_by: 0 } ] } }
    Endorsement:
      version: "0"
      mod_policy: Admins
      policy:
        type: 1
        value:                       # OR('org5.member')
          version: 0
          identities:
            - principal_classification: ROLE
              principal: { msp_identifier: org5, role: MEMBER }
          rule: { n_out_of: { "n": 1, rules: [ { signed_by: 0 } ] } }
  values:
    MSP:
      version: "0"
      mod_policy: Admins
      value:
        type: 0                      # 0 = FABRIC
        config:                      # FabricMSPConfig
          name: org5                 # MSP ID; also the msp_id of the party's consenter
          root_certs:                # the org's CA certificates
            - crypto/org5/ca/ca-cert.pem
          intermediate_certs: []
          admins:                    # the org's admin certificates
            - crypto/org5/users/Admin/msp/signcerts/cert.pem
          revocation_list: []
          signing_identity: null
          organizational_unit_identifiers: []
          crypto_config:
            signature_hash_family: SHA2
            identity_identifier_hash_function: SHA256
          tls_root_certs:            # the org's TLS CA certificates; must be the party's TLSCACerts
            - crypto/org5/tlsca/tlsca-cert.pem
          tls_intermediate_certs: []
          fabric_node_ous: null
          known_certs: []
    Endpoints:
      version: "0"
      mod_policy: Admins
      value:
        addresses:                   # the party's router (broadcast) and assembler (deliver)
          - id=5,msp-id=org5,broadcast,router.org5.com:7050
          - id=5,msp-id=org5,deliver,assembler.org5.com:7060
```

**Flags**

| Flag | Required | Description |
|------|:--------:|-------------|
| `--party` | yes | Path to the party definition YAML |
| `--block` | yes | Config block to edit in place |

**What else changes in the block**

The command also updates the parts of the block that must agree with the new party:

| Element | Change |
|---------|--------|
| Shared config `MaxPartyID` | Set to the new `PartyID`. |
| Consenter mapping (the `Orderers` value) | A new entry for the party's consenter. Its host, port and certificates come from `ConsenterConfig`, and its `msp_id` from the org's MSP `name`. |
| `BlockValidation` policy | Recomputed as a BFT quorum over all the consenters' signing certificates. Its version and `mod_policy` are kept. |

**When the command fails**

The command leaves the block unchanged if:

- `party5.yaml` cannot be parsed, has no `Party` or `Org` section, or `Org` has no `Name`.
- A key of `Party` or `Org` is not a field of the block structure.
- A certificate or CRL file can't be read, or doesn't hold what its field needs (see
  **Certificates**).
- An orderer org with the same `Name` is already in the block. Adding a party to an existing org is
  not supported.
- `Party` has no `ConsenterConfig`, or `Org` has no Fabric MSP with a `name`.
- The `host` of a node in `Party` (router, batcher, consenter or assembler) is not a valid host.
  It must follow the same rules as `modify party node --host`; see **Host format** under
  **modify party node**.
- An address in the org's `Endpoints` value cannot be parsed, or its host is not a valid host
  under the same rules.
- The edited config is not a valid channel config. For example, the org's MSP rejects a known
  certificate that none of its CAs issued.

fxadmin checks the hosts only because the orderer doesn't check them yet. These checks will be
removed from `modify party add` once the orderer checks hosts and fxadmin runs the orderer's checks
before it writes the block ([Chaguar/common#233](https://github.com/Chaguar/common/issues/233)).

**Checked by the orderer on submission**

The orderer checks these rules when the config update is submitted. fxadmin doesn't run them yet
before it writes the block, so a mistake is reported only on submission. Running them in fxadmin is
tracked in [Chaguar/common#233](https://github.com/Chaguar/common/issues/233).

- `PartyID` must be the block's `MaxPartyID + 1`.
- The party's `CACerts` and `TLSCACerts` must be valid CA certificates. Every node's
  `server_tls_cert` must be issued by `TLSCACerts` and its `sign_cert` by `CACerts`, and none may
  be expired.
- The org's MSP `tls_root_certs` must be the party's `TLSCACerts`.
- No two nodes, of this party or another, may have signing certificates with the same public key.
- The org's `Endpoints` must include both a `broadcast` and a `deliver` address.
- A config update can change only one party. Don't add a party and change another one in the same
  `next.pb`.

**Not checked yet**

Neither fxadmin nor the orderer checks these rules yet, so check them yourself:

- Every node has a port between 1 and 65535, and the party has at least one batcher
  ([Chaguar/orderer#1449](https://github.com/Chaguar/orderer/issues/1449)).
- Every address in the org's `Endpoints` has the party's `id=` and exactly one API, `broadcast` or
  `deliver` ([Chaguar/orderer#1477](https://github.com/Chaguar/orderer/issues/1477)).
- Every node's `client_tls_cert` is a valid certificate issued by `TLSCACerts`. will be added
as part of [Chaguar/orderer#1451](https://github.com/Chaguar/orderer/issues/1451)

---

**modify party remove**

Remove an ARMA party and its orderer organization, unless the organization is still associated with another party.

```bash
fxadmin modify party remove 5 --block next.pb
```

**Arguments**

| Argument | Required | Description |
|----------| :------: |-------------|
| `<party-id>` | yes | Numeric `PartyID` to remove |

**Flags**

| Flag | Required | Description |
|------|:--------:|-------------|
| `--block` | yes | Config block to edit in place |

**What changes in the block**

The party's organization is the orderer organization whose MSP `name` is the `msp_id` of the
party's consenter in the `Orderers` value. This is how the orderer links a party to its
organization.

| Element | Change |
|---------|--------|
| Shared config `PartiesConfig` | The party's entry is removed. |
| Shared config `MaxPartyID` | Unchanged, so a removed `PartyID` is never used again. The next party you add still gets `MaxPartyID + 1`. |
| Consenter mapping (the `Orderers` value) | The party's consenter is removed. |
| The party's orderer organization (`channel_group.groups.Orderer.groups.<org>`) | Removed, unless the consenter of another party has the same `msp_id`. In that case the organization is kept and only the party's addresses are removed from its `Endpoints`. |
| `Endpoints` of every remaining orderer organization | Every address with the party's `id=` is removed. Addresses without an `id=` are kept. |
| `BlockValidation` policy | Recomputed as a BFT quorum over the signing certificates of the remaining consenters. Its version and `mod_policy` are kept. |

**When the command fails**

The command leaves the block unchanged if:

- The shared config has no party with that `PartyID`.
- The party is the only party in the shared config. Removing it would leave no ordering service.
- The consenter mapping has no consenter for the party.
- No orderer organization, or more than one, has the MSP `name` of the party's consenter.
- An address in the `Endpoints` value of an orderer organization cannot be parsed.
- The edited config is not a valid channel config.

**Checked by the orderer on submission**

The orderer checks these rules when the config update is submitted. fxadmin doesn't run them yet
before it writes the block, so a mistake is reported only on submission. Running them in fxadmin is
tracked in [Chaguar/common#233](https://github.com/Chaguar/common/issues/233).

- A config update can change only one party. Don't remove a party and also add, remove or change
  another one in the same `next.pb`.
- Every orderer organization that is kept must still have both a `broadcast` and a `deliver`
  address.

**Not checked yet**

Neither fxadmin nor the orderer checks yet that the remaining parties can still reach a BFT
quorum. Make sure the ordering service stays live without the removed party before you submit.

---

**modify party node**

Change one party node's endpoint and/or certificates. The node is selected by
`--party` + `--role` (+ `--shard` for a batcher); the optional field flags say
what to change. Any subset of fields can be changed in one command.

```bash
# move an endpoint
fxadmin modify party node --party 1 --role batcher --shard 1 --port 9014 --block next.pb

# rotate a cert
fxadmin modify party node --party 1 --role router --tls-cert new-router-tls.crt --block next.pb

# several fields of one node at once
fxadmin modify party node --party 1 --role consenter \
  --sign-cert new-c-sign.crt --tls-cert new-c-tls.crt --port 8052 --block next.pb
```

**Flags**

| Flag | Required | Description |
|------|:--------:|-------------|
| `--party` | yes | Party ID |
| `--role` | yes | `router \| batcher \| consenter \| assembler` |
| `--shard` | if batcher | Batcher shard ID. Required with `--role batcher`; an error with any other role |
| `--host` | no | New host: a host name or an IP address, without a port; IPv6 without brackets (unchanged if omitted) |
| `--port` | no | New port, 1–65535 (unchanged if omitted) |
| `--tls-cert` | no | Path to new TLS certificate, set as both the node's client and server TLS certificate (unchanged if omitted) |
| `--sign-cert` | no | Path to new signing certificate (unchanged if omitted) |
| `--block` | yes | Config block to edit in place |

All four node roles have a signing certificate, so `--sign-cert` works with any
`--role`. Changing a `router` or `assembler` endpoint also updates that party's
orderer-organization `Endpoints` value (the router is advertised as the
`broadcast` address, the assembler as the `deliver` address).

**Host format.** `--host` takes an IP address or a host name, without a port:

- An IPv4 address (`10.0.0.5`) or an IPv6 address (`2001:db8::7`). Give IPv6 without
  brackets; the `Endpoints` value adds them, for example `[2001:db8::7]:7050`. An IPv6
  zone (`fe80::1%eth0`) is rejected.
- A host name: labels joined by `.`, each of up to 63 ASCII letters, digits and `-`, not
  starting or ending with `-`, and at most 253 characters in total. `_` and non-English
  letters are rejected, for example `orderer_1.example.com`.
- Digits and dots that are not a valid IPv4 address, such as `300.1.1.1`, are rejected.

A rejected host leaves the block unchanged.

___

**modify party ca add | remove | set | node-ous**

Change a party's sign-CA (`--sign-cert`) and/or TLS-CA (`--tls-cert`) certificate
**lists**, or the CA that the node OUs of the party's org MSP pin.

```bash
# add a new CA alongside the current ones
fxadmin modify party ca add    --party 1 --sign-cert new-ca.crt --block next.pb

# remove a retired TLS-CA
fxadmin modify party ca remove --party 1 --tls-cert old-tlsca.crt --block next.pb

# replace both lists
fxadmin modify party ca set --party 1 --sign-cert new-ca.crt --tls-cert new-tlsca.crt --block next.pb

# point the node OUs at a CA of the party's CA list
fxadmin modify party ca node-ous --party 1 --sign-cert new-ca.crt --block next.pb
```

**Subcommands**

| Subcommand | Effect |
|------------|--------|
| `add`      | Append the given cert(s) to the named list(s). A cert that is already in the list is skipped with a message, so old and new CAs coexist. |
| `remove`   | Drop the given cert(s) from the named list(s). Fails if a cert is not in the list, if the list would be left empty, or if it would drop the CA the node OUs pin. |
| `set`      | Replace the whole named list(s). Whatever CA(s) a list holds, one or more, it then holds exactly the given cert(s), one or more, in the order given. Fails if it would drop the CA the node OUs pin. |
| `node-ous` | Point the org's node OUs at the given CA, which must already be in the party's CA list. The lists don't change. This is the only command that moves the node OUs. |

**Flags**

| Flag | Required | Description |
|------|:--------:|-------------|
| `--party` | yes | Party whose CA list(s) change |
| `--sign-cert` | ≥1 of the two | PEM path(s) for the signing-CA list (repeatable). `node-ous`: required, exactly one |
| `--tls-cert` | ≥1 of the two | PEM path(s) for the TLS-CA list (repeatable). Not accepted by `node-ous` |
| `--block` | yes | Config block to edit in place |

Pass only `--sign-cert` to change the CA list, only `--tls-cert` for the TLS-CA
list, or both to change both. Each file must hold exactly one PEM-encoded X.509 **CA** certificate,
and the same certificate can't be given twice. A file of several certificates, such as a CA chain
or bundle, is rejected: give each CA in a file of its own.

**What changes in the block**

Each list is stored in two places, and the command changes both:

| List | Party (`channel_group.groups.Orderer.values.ConsensusType.value.metadata.PartiesConfig[]`) | Orderer org MSP (`channel_group.groups.Orderer.groups.<org>.values.MSP.value.config`) |
|------|------|------|
| `--sign-cert` | `CACerts` | `root_certs` |
| `--tls-cert` | `TLSCACerts` | `tls_root_certs` |

Intermediate CAs aren't supported yet ([Chaguar/orderer#619](https://github.com/Chaguar/orderer/issues/619)):
a certificate always goes to `root_certs` / `tls_root_certs`, and `intermediate_certs` /
`tls_intermediate_certs` are left unchanged.

The party's org is the orderer org whose MSP `name` is the `msp_id` of the party's consenter.
An org can run several parties, which all trust the org's MSP, so the change is applied to **every
party of the org**, whichever of them `--party` names. The orderer accepts only one changed party
per config update, so today it rejects a CA change of an org that runs several parties.

The orderer org is assumed not to be an application org too (see
[Modify Commands](#modify-commands-automation-of-the-manual-step)): only its MSP in the `Orderer`
group changes.

**Node OUs.** Each node OU of the org MSP (`fabric_node_ous`: client, peer, admin and orderer)
pins one CA, which must be one of the MSP's `root_certs` or `intermediate_certs`. Every identity of
the org, including the party's nodes, must be issued by the CA its node OU pins. So run `node-ous`
in the same config update as `modify party node --sign-cert` for every node of the org's parties
(see the rotation below).

**Node OUs that are not enabled.** An MSP can define node OUs with `enable: false`. Then they don't
decide the roles of identities, but the MSP still requires the CA each node OU pins to be one of its
`root_certs` or `intermediate_certs`. So before you remove the CA they pin (with `ca remove` or
`ca set`), you still move the pins with `fxadmin modify party ca node-ous`. It moves them as usual,
and warns that the node OUs are not enabled. `ca add`, `ca remove` and `ca set` never move the pins.

Each command changes one thing. `add`, `remove` and `set` change only the CA lists, and
`node-ous` changes only the node OUs: it points every node OU that pins a CA at the given CA. So
`remove` and `set` never move the node OUs. If either would drop the CA they pin, it fails before
it writes the block, and tells you to move them first:

```
the node OUs pin the CA "CN=org1-CA,…", which is not in the CA list; first point them at a CA of
the list, with `fxadmin modify party ca node-ous`
```

**Rotating a party's CAs**

Replacing a party's CAs takes **three config updates**, as in the orderer's
`TestChangePartyCACertificates` scenario and its
[CA rotation guide](https://github.com/Chaguar/orderer/blob/main/docs/reconfig/reconfig.md#55-rotate-a-partys-ca-certificates).

**The rotation assumes an org MSP with node OUs.** Then the admin role comes from the admin node OU,
and the MSP's `admins` list is empty. This is the default for orgs generated with node OUs.

**MSPs without node OUs are not supported.** An MSP without node OUs (admincerts mode) lists its
admin certificates in `admins` instead, and has no node OU pins to move. To rotate the CAs of such
an org, update 2 should replace the `admins` certificates with those of the new admin identities,
instead of running `node-ous`. The `modify` commands don't change `admins`, and won't: CA rotation
with the `modify` commands is supported only for MSPs with node OUs. The same holds for an MSP with
node OUs that also lists admin certificates in `admins`: `ca set` in update 3 fails while they are
issued by the old CA.

Each update removes trust in the old CA only once nothing in the block depends on it:

| Update | `root_certs` / `tls_root_certs` | Party's node certificates | Node OUs pin | Party's admin endorses with |
|--------|------|------|------|------|
| 1. Trust the new CAs | old, new | old | old | old identity |
| 2. Move the nodes and the node OUs to the new CA | old, new | **new** | **new** | old identity |
| 3. Remove the old CAs | **new** | new | new | **new** identity |

The node certificates and the node OUs must move **in the same update**, update 2. A node
certificate that isn't issued by the CA its node OU pins is invalid in the org's MSP, so moving
only one of them is rejected on submission.

The org MSP's `known_certs` follow the identities. A client that signs with the ID of its
certificate is resolved only through `known_certs`. So update 2, where the clients switch to their
new identities, adds the new CA's certificates to `known_certs`, and update 3 sets `known_certs` to
the new CA's certificates only, which drops the old ones.

**It is the admin's responsibility to follow this order:** before `set` removes a CA, no certificate
of the party may still be issued by it.

**Before update 1**, issue from the new CAs a TLS and a signing certificate for every node of the
party (the router, the consenter, the assembler, and the batcher of every shard), and new admin
and client identities.

**Recipe: the full CA rotation**

The commands below rotate the CAs of party 1 of `org1`. `config0.pb` is the latest config block.
`admin_orgN.yaml` is the admin configuration of `orgN`, and `admin_org1_new.yaml` is `org1`'s
admin configuration with its identity from the new CA. Each update is endorsed by a majority of the
orderer orgs, including `org1`.

```bash
# ---- Update 1: trust the new CAs alongside the old ones.
cp config0.pb next.pb
fxadmin modify party ca add --party 1 --sign-cert new-ca.crt --tls-cert new-tlsca.crt --block next.pb
fxadmin compute-update --pb config0.pb next.pb --output update1.pb
fxadmin tx endorse update1.pb --config admin_org1.yaml --output update1_org1.pb      # org1: old identity
fxadmin tx endorse update1.pb --config admin_org2.yaml --output update1_org2.pb
fxadmin tx endorse update1.pb --config admin_org3.yaml --output update1_org3.pb
fxadmin tx merge update1_org1.pb update1_org2.pb update1_org3.pb --output update1_endorsed.pb
fxadmin tx send update1_endorsed.pb --config admin_org2.yaml --current-block config0.pb --output update1_tx.pb
fxadmin follow --config admin_org2.yaml --current-block config0.pb --timeout 60s --output config1.pb

# ---- Update 2: move every node of party 1, and the node OUs, to the new CA.
cp config1.pb next.pb
fxadmin modify party node --party 1 --role router    --tls-cert router-tls.crt    --sign-cert router-sign.crt    --block next.pb
fxadmin modify party node --party 1 --role consenter --tls-cert consenter-tls.crt --sign-cert consenter-sign.crt --block next.pb
fxadmin modify party node --party 1 --role assembler --tls-cert assembler-tls.crt --sign-cert assembler-sign.crt --block next.pb
fxadmin modify party node --party 1 --role batcher --shard 1 --tls-cert batcher1-tls.crt --sign-cert batcher1-sign.crt --block next.pb
#   … a batcher command for every shard of party 1 …
fxadmin modify party ca node-ous --party 1 --sign-cert new-ca.crt --block next.pb
#   Add the new CA's certificates to org1's MSP known_certs, so that clients that sign with the ID
#   of their new certificate are resolved once they switch. List the batcher of every shard of
#   party 1, not only batcher1 (batcher2-sign.crt, …).
fxadmin modify orderer known-certs add --org org1 \
  --cert admin-sign.crt --cert client-sign.crt --cert router-sign.crt \
  --cert consenter-sign.crt --cert assembler-sign.crt --cert batcher1-sign.crt --block next.pb
fxadmin compute-update --pb config1.pb next.pb --output update2.pb
fxadmin tx endorse update2.pb --config admin_org1.yaml --output update2_org1.pb      # org1: old identity
fxadmin tx endorse update2.pb --config admin_org2.yaml --output update2_org2.pb
fxadmin tx endorse update2.pb --config admin_org3.yaml --output update2_org3.pb
fxadmin tx merge update2_org1.pb update2_org2.pb update2_org3.pb --output update2_endorsed.pb
fxadmin tx send update2_endorsed.pb --config admin_org2.yaml --current-block config1.pb --output update2_tx.pb
fxadmin follow --config admin_org2.yaml --current-block config1.pb --timeout 60s --output config2.pb
#   Now restart party 1's nodes with their new certificates and keys, and switch party 1's admin
#   and clients to their identities from the new CA.

# ---- Update 3: remove the old CAs.
cp config2.pb next.pb
#   First, set org1's MSP known_certs to the new CA's certificates only, which drops the old CA's.
#   List the batcher of every shard of party 1, not only batcher1 (batcher2-sign.crt, …): set keeps
#   only the certificates it is given, and an old-CA certificate left in known_certs makes the
#   `ca set` below fail.
fxadmin modify orderer known-certs set --org org1 \
  --cert admin-sign.crt --cert client-sign.crt --cert router-sign.crt \
  --cert consenter-sign.crt --cert assembler-sign.crt --cert batcher1-sign.crt --block next.pb
fxadmin modify party ca set --party 1 --sign-cert new-ca.crt --tls-cert new-tlsca.crt --block next.pb
fxadmin compute-update --pb config2.pb next.pb --output update3.pb
fxadmin tx endorse update3.pb --config admin_org1_new.yaml --output update3_org1.pb  # org1: new identity
fxadmin tx endorse update3.pb --config admin_org2.yaml --output update3_org2.pb
fxadmin tx endorse update3.pb --config admin_org3.yaml --output update3_org3.pb
fxadmin tx merge update3_org1.pb update3_org2.pb update3_org3.pb --output update3_endorsed.pb
fxadmin tx send update3_endorsed.pb --config admin_org2.yaml --current-block config2.pb --output update3_tx.pb
fxadmin follow --config admin_org2.yaml --current-block config2.pb --timeout 60s --output config3.pb
```

**When the command fails**

The command leaves the block unchanged if:

- **Every subcommand:**
  - a certificate file can't be read, isn't a PEM-encoded X.509 CA certificate, holds more than
    one PEM block, or is given twice;
  - the shared config has no party with that `PartyID`, or the consenter mapping has no consenter
    for it;
  - no orderer org, or more than one, has the MSP `name` of the party's consenter;
  - the edited config is not a valid channel config.
- **`remove`:**
  - a certificate is not in the list of every party of the org;
  - a list would be left empty;
  - the CA list would no longer hold the CA the node OUs pin. Run `node-ous` first;
  - the org MSP still has an admin certificate (`admins`) or a known certificate (`known_certs`)
    issued by a CA it removes.
- **`set`:**
  - the CA list would no longer hold the CA the node OUs pin. Run `node-ous` first;
  - the org MSP still has an admin certificate (`admins`) or a known certificate (`known_certs`)
    issued by a CA it removes.
- **`node-ous`:**
  - not exactly one `--sign-cert` is given, or a `--tls-cert` is given;
  - the CA is not in the CA list of every party of the org, or not in the org MSP's `root_certs`
    or `intermediate_certs`;
  - the org MSP has no node OUs that pin a CA.

`add` fails only in the cases of every subcommand.

---

### Compute a config update Command

Compute the **`ConfigUpdate`** which is the delta between the original configuration and the
modified configuration.

`compute-update` has two modes for supplying that pair:

- **JSON mode** (default) - compute the update from two config **JSON** files. Goes with the [manual step](#manual-step).
- **Block mode** (`--pb`) - compute the update from two config **block** files. Goes with the
  [Modify Commands](#modify-commands-automation-of-the-manual-step), which edit a
  block directly.

Both modes produce the same kind of `ConfigUpdate`. If the two inputs are identical the command
produces an empty update and reports that there is nothing to do.

<br><br>
#### JSON mode (default)

The original is the config JSON decoded from `--current-block`; the modified is its edited copy.
The channel ID is taken from the current config block supplied with `--current-block`
and stamped onto the resulting `ConfigUpdate`. All three inputs must belong to the
**same channel**.

```bash
fxadmin compute-update <current.json> <modified.json> --current-block <current_block.pb> --output <config_update.pb>
```

Example:
```bash
fxadmin compute-update \
   current_config.json \
   modified_config.json \
   --current-block last_config.pb \
   --output config_update.pb
```

**Arguments** (order matters: original first, modified second)

| Argument          | Required | Description                                            |
|-------------------| :------: |--------------------------------------------------------|
| `<current.json>`  |   yes    | Original configuration, as decoded from `--current-block` |
| `<modified.json>` |   yes    | Modified configuration (edited copy of `current.json`) |

**Flags**

| Flag              | Required | Description                                                          |
|-------------------| :------: |----------------------------------------------------------------------|
| `--current-block` |   yes    | Path to the current config block whose channel ID the update targets |
| `--output`        |   yes    | Path to the output `ConfigUpdate` protobuf file                      |


<br><br>
#### Block mode (`--pb`)

Computes the delta directly from two config **block** files.
This command is for use with the [Modify Commands](#modify-commands-automation-of-the-manual-step): 
the `modify` commands edit a block file directly, so the delta is computed from the
original block and the edited ("next") block.


```bash
fxadmin compute-update --pb <current.pb> <next.pb> --output <config_update.pb>
```

Example:
```bash
fxadmin compute-update --pb \
   current.pb \
   next.pb \
   --output config_update.pb
```

**Arguments** (order matters: original first, modified second)

| Argument       | Required | Description                                                          |
|----------------| :------: |----------------------------------------------------------------------|
| `<current.pb>` |   yes    | **Original** config block - the one fetched from the ledger          |
| `<next.pb>`    |   yes    | **Modified** config block - the copy edited by the `modify` commands |

**Flags**

| Flag       | Required | Description                                                          |
|------------| :------: |----------------------------------------------------------------------|
| `--pb`     |   yes    | Select block mode (diff two config blocks instead of two JSON files) |
| `--output` |   yes    | Path to the output `ConfigUpdate` protobuf file                      |

---
### Transaction Command

The tx command endorses, merges, prepares, and submits configuration update transactions.

```bash
fxadmin tx <command> [command arguments] [command flags]
```

#### Commands

```bash
# Endorse a configuration update
fxadmin tx endorse config_update.pb \
  --config admin.yaml \
  --output endorsed_config_update1.pb

# Merge multiple endorsements
fxadmin tx merge \
  endorsed_config_update1.pb \
  endorsed_config_update2.pb \
  --output endorsed_config_update.pb

# Prepare the configuration transaction
fxadmin tx prepare endorsed_config_update.pb \
  --config admin.yaml \
  --output config_tx.pb

# Submit the configuration transaction
fxadmin tx submit config_tx.pb \
  --config admin.yaml \
  --current-block current_block.pb
  
# Prepare + submit the configuration transaction
fxadmin tx send endorsed_config_update.pb \
   --config admin.yaml \
   --current-block current_block.pb \
   --output config_tx.pb
```
---

#### Endorse a configuration update

The endorse command signs a configuration update using the administrator identity defined in the admin configuration YAML file.  
The admin signs on `ConfigUpdate`, producing an endorsed update that
carries that admin's `ConfigSignature`.

```bash
fxadmin tx endorse \
  <config_update.pb> \
  --config <admin.yaml> \
  --output <endorsement.pb>
```
**Arguments**

| Argument             | Required | Description                                    |
|----------------------| :------: | ---------------------------------------------- |
| `<config_update.pb>` |   yes    | Path to the configuration update protobuf file to endorse |

**Flags**

| Flag         | Required | Description                                                       |
| ------------ | :------: | ---------------------------------------------------------------- |
| `--config`   |   yes    | Path to the admin configuration YAML file containing the administrator signing identity |
| `--output`   |   yes    | Path to the generated endorsement protobuf file    |

**Example**

```bash
# org1 admin endorses
fxadmin tx endorse config_update.pb \
  --config admin_org1.yaml \
  --output endorsed_config_update1.pb

# org2 admin endorses
fxadmin tx endorse config_update.pb \
  --config admin_org2.yaml \
  --output endorsed_config_update2.pb
```

---

#### Merge Endorsements
The merge command combines multiple endorsements into a single endorsed configuration update envelope (`ConfigUpdateEnvelope`).
Use this to gather the endorsements needed to satisfy the channel's admin policy (e.g. a majority of orderer orgs). 
Duplicate signers are de-duplicated.

```bash
fxadmin tx merge \
  <endorsement.pb>... \
  --output <endorsed_config_update.pb>
```
**Arguments**

| Argument              | Required | Description                                          |
| --------------------- | :------: | ---------------------------------------------------- |
| `<endorsement.pb>...`  |   yes    | Paths to one or more endorsement protobuf files to merge   |

**Flags**

| Flag         | Required | Description                                                  |
| ------------ | :------: | ------------------------------------------------------------ |
| `--output`   |   yes    | Path to the merged configuration update envelope |

**Example**

```bash
fxadmin tx merge \
  endorsed_config_update1.pb \
  endorsed_config_update2.pb \
  --output endorsed_config_update.pb
```

> **Note:** The `merge` command is only required when the configuration update has been endorsed by multiple administrators. If a single administrator endorsement satisfies the endorsement policy, the output of `tx endorse` can be used directly as input to `tx prepare` or `tx send`.

---

#### Prepare a Configuration Transaction

The prepare command creates a ready to submit configuration transaction envelope from the endorsed configuration update.
The resulting transaction is signed by the submitting client which is a channel writer (it can be the admin).

```bash
fxadmin tx prepare \
  <endorsed_config_update.pb> \
  --config <admin.yaml> \
  --output <config_tx.pb>
```

**Arguments**

| Arguments                                  | Required | Description                                                            |
|---------------------------------------| :------: | --------------------------------------------------------------------- |
| `<endorsed_config_update.pb>` |   yes    | Path to the endorsed configuration update protobuf file  |


**Flags**

| Flag         | Required | Description                                                             |
| ------------ | :------: | ---------------------------------------------------------------------- |
| `--config`   |   yes    | Path to the admin configuration YAML file containing the submitting client identity  |
| `--output`   |   yes    | Path to the generated configuration transaction protobuf file                   |

**Example**
```bash
fxadmin tx prepare endorsed_config_update.pb \
  --config admin.yaml \
  --output config_tx.pb
```

---

#### Submit a Configuration Transaction

The submit command submits a prepared configuration transaction to all routers via the
Broadcast API. A router forwards it into the ordering pipeline; once ordered and
committed, the new configuration takes effect across all parties.

The command broadcasts to every router and collects each router's acknowledgement, logging
the per-router outcome. It **succeeds** only when a BFT quorum of the routers acknowledges 
the transaction: with `n` parties (read from the`--current-block`) the quorum is `2f+1`, 
where `f = (n-1)/3` is the number of faulty parties the network tolerates. 
If fewer than a quorum acknowledge — because routers rejected the
transaction or were unreachable — the command fails, so a partial or failed delivery is
reported rather than silently succeeding.

```bash
fxadmin tx submit \
  <config_tx.pb> \
  --config <admin.yaml> \
  --current-block <current_block.pb>
```

**Arguments**

| Argument       | Required | Description                                     |
|----------------| :------: | ----------------------------------------------- |
| `<config_tx.pb>` |   yes    | Path to the prepared configuration transaction protobuf file       |

**Flags**

| Flag              | Required | Description                                                               |
| ----------------- | :------: |---------------------------------------------------------------------------|
| `--current-block` |   yes    | Path to the current block protobuf file containing the router's endpoints |
| `--config`        |   yes    | Path to the admin configuration YAML file                                 |

**Example**

```bash
fxadmin tx submit config_tx.pb \
  --config admin.yaml \
  --current-block current_block.pb
```

After submitting the transaction, confirm that the configuration update has been committed by reading the latest configuration block until the updated configuration block is available.

---
#### Send a Configuration Update
The send command prepares and submits an endorsed configuration update in a single step.

It creates a configuration transaction from the endorsed configuration update, signs it using the submitting client identity defined in the admin configuration YAML file, and submits the transaction to all configured routers.

The send command is equivalent to running `prepare` followed by `submit`. As with `submit`, it
succeeds only when a BFT quorum (`2f+1`) of the routers acknowledged the transaction, and fails
otherwise.

```bash
fxadmin tx send \
 <endorsed_config_update.pb> \
 --config <admin.yaml> \
 --current-block <current_block.pb> \
 --output <config_tx.pb>
```
**Arguments**

| Arguments                                  | Required | Description                                                            |
|---------------------------------------| :------: | --------------------------------------------------------------------- |
| `<endorsed_config_update.pb>` |   yes    | Path to the endorsed configuration update protobuf file  |


**Flags**

| Flag         | Required | Description                                                             |
| ------------ | :------: | ---------------------------------------------------------------------- |
| `--current-block` |   yes    | Path to the current block protobuf file containing the router's endpoints |
| `--config`        |   yes    | Path to the admin configuration YAML file                                 |
| `--output`        |   yes    | Path to write the prepared configuration transaction to, for record keeping |

The prepared configuration transaction is written to `--output` before it is broadcast, so a record of what was submitted is kept even if the broadcast fails.

**Example**
```bash
fxadmin tx send \
   endorsed_config_update.pb \
   --config admin.yaml \
   --current-block current_block.pb \
   --output config_tx.pb
```
---
### Follow the assembler ledger
The follow command waits for the next config block to commit across all assemblers.

It reads the current config block to learn the current config sequence `S` and the assembler
endpoints, and waits for the block a pending update will produce once committed:
`expected = S + 1`. A configuration update changes at most one assembler, so the endpoints in the
current block still reach the rest; assemblers that cannot be reached are reported as unreachable.

For each assembler, follow pulls blocks until it observes a config block whose sequence is
`expected` (or higher — a later update may also have committed), which means the next config is
committed on that assembler. An assembler whose last config sequence is still below `expected` is
behind (the config is not committed there yet). Polling continues per assembler until it commits
or the timeout expires. When done, the command prints, for each assembler, its last block number,
the last config sequence in its ledger, and whether it committed.

The `--timeout` is the hard upper bound on the whole command, so an unresponsive assembler cannot block
past it. An assembler that never answers within the window is reported as `unreachable`.

Once at least `f+1` assemblers report the **same** next config block (`f = (n-1)/3`
is the number of faulty parties the network of `n` parties tolerates), that block is written
to `--output`. The written block is ready to serve as the `--current-block` of the next
reconfiguration round. If a quorum of `f+1` matching blocks is not reached before the timeout,
follow prints the report, writes no output file, and exits with an error.

Blocks are compared by their **header and data only**, ignoring the block metadata, which
carries the orderer signatures over the block. In ARMA, block metadata may contain different signer sets 
or signature ordering, so the raw block bytes may differ.

```bash
fxadmin follow \
   --config <admin.yaml> \
   --current-block <current_block.pb> \
   --timeout <duration> \
   --output <next_config.pb>
```

**Flags**

| Flag              | Required | Description                                                                             |
|-------------------| :------: |-----------------------------------------------------------------------------------------|
| `--current-block` |   yes    | Path to the current block protobuf file containing the assembler's endpoints            |
| `--config`        |   yes    | Path to the admin configuration YAML file                                               |
| `--timeout`       |   yes    | Maximum amount of time to pull blocks from the assemblers before reporting the results  |
| `--output`        |   yes    | Path to write the next committed config block to, for use as the next `--current-block` |

**Example**
```bash
fxadmin follow \
 --config admin.yaml \
 --current-block last_config.pb \
 --timeout 30s \
 --output next_config.pb
```
**Output**

The command logs a one-line summary, for example:

```
expected last config sequence: 5, 3 out of 4 assemblers committed a block with last config sequence 5
```

If any assembler has not committed when the timeout elapses, it also logs a warning, for example:

```
timeout of 30s elapsed with 1 of 4 assemblers not yet committed to last config sequence 5
```

It then prints the polling timeout and a per-assembler table:

```
polling timeout: 30s
ASSEMBLER        LAST BLOCK  LAST CONFIG SEQUENCE  STATUS
assembler1:7051  104         5                     committed
assembler2:7053  104         5                     committed
assembler3:7055  104         5                     committed
assembler4:7057  103         4                     behind
```

An assembler that could not be reached during the whole window is shown with `-` for both its last
block and last config sequence, and an `unreachable` status.

When at least f+1 assemblers agrees on the next config block, follow writes it to `--output` and logs, for example:

```
config block at last config sequence 5 agreed by 3 assemblers (quorum 3), written to next_config.pb
```

If less than f+1 agrees before the timeout, no output file is written and the command fails with an error
such as `no config block at last config sequence 5 was agreed by a quorum of 3 assemblers`.

 ---

## End-to-end walkthrough (with Manual step)

### Single organization

When one organization's admin signature satisfies the config policy, the whole
flow can be run by that admin:

```bash
# 1. Read the current configuration.
fxadmin ledger --config=admin.yaml --current-block=current_block.pb config latest --output last_config.pb

# 2. Decode it to JSON.
fxadmin decode last_config.pb --output=current_config.json

# 3. Edit manually: copy the current_config.json to modified_config.json and edit.

# 4. Compute the update.
fxadmin compute-update current_config.json modified_config.json --current-block=last_config.pb --output=config_update.pb

# 5. Endorse, prepare, submit.
fxadmin tx endorse config_update.pb --config=admin.yaml --output=endorsed_config_update.pb
fxadmin tx prepare endorsed_config_update.pb --config=admin.yaml --output=config_tx.pb
fxadmin tx submit  config_tx.pb --config=admin.yaml --current-block=last_config.pb

# 6. Follow the assembler ledger to make sure the config tx was committed.
#    The new config block is written to next_config.pb, ready to be
#    the --current-block of the next reconfiguration round.
fxadmin follow --config=admin.yaml --current-block=last_config.pb --timeout=60s --output=next_config.pb

```

### Multiple organizations

When the config policy requires signatures from more than one organization, each admin
endorses the same `ConfigUpdate` independently, the endorsements are merged, and a single
submitting client (any channel writer, e.g. the org1 admin) prepares and submits the
transaction:

```bash
# 1. Read the current configuration.
fxadmin ledger --config=admin_org1.yaml --current-block=current_block.pb config latest --output last_config.pb

# 2. Decode it to JSON.
fxadmin decode last_config.pb --output=current_config.json

# 3. Edit manually: copy the current_config.json to modified_config.json and edit.

# 4. Compute the update.
fxadmin compute-update current_config.json modified_config.json --current-block=last_config.pb --output=config_update.pb

# 5. Each org endorses the same update independently.
fxadmin tx endorse config_update.pb --config=admin_org1.yaml --output=endorsed_config_update1.pb
fxadmin tx endorse config_update.pb --config=admin_org2.yaml --output=endorsed_config_update2.pb

# 6. Merge the endorsements into a single endorsed config update.
fxadmin tx merge endorsed_config_update1.pb endorsed_config_update2.pb --output=endorsed_config_update.pb

# 7. Prepare and submit in one step with send (prepare + submit).
fxadmin tx send endorsed_config_update.pb --config=admin_org1.yaml --current-block=last_config.pb --output=config_tx.pb

# 8. Follow the assembler ledger to make sure the config tx was committed.
#    The new config block is written to next_config.pb, ready to be
#    the --current-block of the next reconfiguration round.
fxadmin follow --config=admin_org1.yaml --current-block=last_config.pb --timeout=60s --output=next_config.pb
```

## End-to-end walkthrough (with Modify commands)

### Multiple organizations
This example changes party 1's batcher (shard 1) endpoint and rotates the router's
TLS certificate - two changes accumulated into one config update:

```bash
# 1. Read the current config block, and copy it to a "next" block.
fxadmin ledger --config=admin_org1.yaml --current-block=current_block.pb config latest --output current.pb
cp current.pb next.pb

# 2. Apply the changes to next.pb (each edits the block in place; changes accumulate).
fxadmin modify party node --party 1 --role batcher --shard 1 --port 9014 --block next.pb
fxadmin modify party node --party 1 --role router --tls-cert new-router-tls.crt --block next.pb

# 3. Compute the update from the two blocks (block mode).
fxadmin compute-update --pb current.pb next.pb --output=config_update.pb

# 4. Each org endorses the same update independently.
fxadmin tx endorse config_update.pb --config=admin_org1.yaml --output=endorsed_config_update1.pb
fxadmin tx endorse config_update.pb --config=admin_org2.yaml --output=endorsed_config_update2.pb
fxadmin tx endorse config_update.pb --config=admin_org3.yaml --output=endorsed_config_update3.pb

# 5. Merge the endorsements into a single endorsed config update.
fxadmin tx merge \
  endorsed_config_update1.pb endorsed_config_update2.pb endorsed_config_update3.pb \
  --output=endorsed_config_update.pb

# 6. Prepare and submit in one step with send (prepare + submit).
fxadmin tx send endorsed_config_update.pb --config=admin_org1.yaml --current-block=current.pb --output=config_tx.pb

# 7. Follow the assembler ledger to make sure the config tx was committed.
fxadmin follow --config=admin_org1.yaml --current-block=current.pb --timeout=60s --output=next_config.pb
```

Only steps 1–3 differ from the manual walkthrough; endorse / merge / send / follow
are identical. For the full command reference see
[Modify Commands](#modify-commands-automation-of-the-manual-step).