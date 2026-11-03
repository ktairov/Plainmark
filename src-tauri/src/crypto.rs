use argon2::{Algorithm, Argon2, Params, Version};
use base64::{engine::general_purpose::URL_SAFE_NO_PAD, Engine as _};
use chacha20poly1305::{
    aead::{Aead, AeadCore, KeyInit, OsRng, Payload},
    XChaCha20Poly1305, XNonce,
};
use zeroize::{Zeroize, Zeroizing};

use crate::error::{AppError, AppResult};

// COMPATIBILITY CONTRACT: every encrypted payload carries its format version. Before the
// first production use, v1's AAD was renamed once from the old working title to Plainmark.
// From this point forward the prefix, KDF parameters, Plainmark AAD, field layout, and
// decrypt_v1 implementation are immutable. A future cipher/KDF change must add enc:v2 plus
// new encrypt_v2/decrypt_v2 functions and a new dispatch arm; never modify v1 again.
const V1_PREFIX: &str = "enc:v1";
const V1_AAD: &[u8] = b"plainmark:encrypted-block:v1";
const V1_SALT_LEN: usize = 16;
const V1_NONCE_LEN: usize = 24;

fn derive_v1_key(password: &str, salt: &[u8]) -> AppResult<[u8; 32]> {
    let params = Params::new(64 * 1024, 3, 4, Some(32))
        .map_err(|_| AppError::new("encryptionSetupFailed", "Could not configure password encryption."))?;
    let argon2 = Argon2::new(Algorithm::Argon2id, Version::V0x13, params);
    let mut key = [0u8; 32];
    argon2
        .hash_password_into(password.as_bytes(), salt, &mut key)
        .map_err(|_| AppError::new("encryptionKeyFailed", "Could not derive an encryption key."))?;
    Ok(key)
}

fn encrypt_v1(
    plaintext: &[u8],
    password: &str,
    salt: &[u8],
    nonce: &[u8],
) -> AppResult<String> {
    if salt.len() != V1_SALT_LEN || nonce.len() != V1_NONCE_LEN {
        return Err(AppError::new("encryptionSetupFailed", "Could not configure encryption randomness."));
    }
    let key = Zeroizing::new(derive_v1_key(password, salt)?);
    let cipher = XChaCha20Poly1305::new_from_slice(&key[..])
        .map_err(|_| AppError::new("encryptionSetupFailed", "Could not initialize encryption."))?;
    let encrypted = cipher
        .encrypt(
            XNonce::from_slice(nonce),
            Payload {
                msg: plaintext,
                aad: V1_AAD,
            },
        )
        .map_err(|_| AppError::new("encryptionFailed", "Could not encrypt the selected text."))?;
    Ok(format!(
        "{V1_PREFIX}:{}:{}:{}",
        URL_SAFE_NO_PAD.encode(salt),
        URL_SAFE_NO_PAD.encode(nonce),
        URL_SAFE_NO_PAD.encode(encrypted)
    ))
}

pub(crate) fn encrypt_text_value(plaintext: String, password: String) -> AppResult<String> {
    let plaintext = Zeroizing::new(plaintext);
    let password = Zeroizing::new(password);
    if plaintext.is_empty() {
        return Err(AppError::new("emptySelection", "Select some text to encrypt."));
    }
    if password.is_empty() {
        return Err(AppError::new("passwordRequired", "Password cannot be empty."));
    }

    let salt_source = XChaCha20Poly1305::generate_nonce(&mut OsRng);
    let salt = &salt_source[..V1_SALT_LEN];
    let nonce = XChaCha20Poly1305::generate_nonce(&mut OsRng);
    encrypt_v1(plaintext.as_bytes(), &password, salt, &nonce)
}

fn decrypt_v1(salt: &str, nonce: &str, ciphertext: &str, password: &str) -> AppResult<String> {
    let salt = URL_SAFE_NO_PAD
        .decode(salt)
        .map_err(|_| AppError::new("invalidEncryptedBlock", "This encrypted block is damaged."))?;
    let nonce = URL_SAFE_NO_PAD
        .decode(nonce)
        .map_err(|_| AppError::new("invalidEncryptedBlock", "This encrypted block is damaged."))?;
    let ciphertext = URL_SAFE_NO_PAD
        .decode(ciphertext)
        .map_err(|_| AppError::new("invalidEncryptedBlock", "This encrypted block is damaged."))?;
    if salt.len() != V1_SALT_LEN || nonce.len() != V1_NONCE_LEN {
        return Err(AppError::new("invalidEncryptedBlock", "This encrypted block is damaged."));
    }

    let key = Zeroizing::new(derive_v1_key(password, &salt)?);
    let cipher = XChaCha20Poly1305::new_from_slice(&key[..])
        .map_err(|_| AppError::new("encryptionSetupFailed", "Could not initialize decryption."))?;
    let decrypted = cipher
        .decrypt(
            XNonce::from_slice(&nonce),
            Payload {
                msg: &ciphertext,
                aad: V1_AAD,
            },
        )
        .map_err(|_| AppError::new("wrongPasswordOrDamaged", "Incorrect password or the encrypted block is damaged."))?;
    match String::from_utf8(decrypted) {
        Ok(text) => Ok(text),
        Err(error) => {
            let mut bytes = error.into_bytes();
            bytes.zeroize();
            Err(AppError::new("invalidDecryptedText", "The decrypted block is not valid text."))
        }
    }
}

pub(crate) fn decrypt_text_value(payload: String, password: String) -> AppResult<String> {
    let password = Zeroizing::new(password);
    if password.is_empty() {
        return Err(AppError::new("passwordRequired", "Password cannot be empty."));
    }
    let parts: Vec<_> = payload.trim().split(':').collect();
    match parts.as_slice() {
        ["enc", "v1", salt, nonce, ciphertext] => decrypt_v1(salt, nonce, ciphertext, &password),
        _ => Err(AppError::new("unsupportedEncryptedFormat", "This encrypted block has an unsupported format.")),
    }
}

#[tauri::command]
pub(crate) fn encrypt_text(plaintext: String, password: String) -> AppResult<String> {
    encrypt_text_value(plaintext, password)
}

#[tauri::command]
pub(crate) fn decrypt_text(payload: String, password: String) -> AppResult<String> {
    decrypt_text_value(payload, password)
}

#[cfg(test)]
mod tests {
    use super::*;

    const V1_FIXTURE: &str = "enc:v1:AAECAwQFBgcICQoLDA0ODw:EBESExQVFhcYGRobHB0eHyAhIiMkJSYn:kjoOIIaoKTbcrT789T4USc1ZP4jD-siNCzoQtC-I0x8dWFRt2OBpKg";

    #[test]
    fn v1_encryption_keeps_its_versioned_wire_format() {
        let salt: Vec<u8> = (0..V1_SALT_LEN as u8).collect();
        let nonce: Vec<u8> = (16..(16 + V1_NONCE_LEN as u8)).collect();
        let encrypted =
            encrypt_v1(b"# Legacy secret\nold note", "archive-pass", &salt, &nonce).unwrap();
        assert_eq!(encrypted, V1_FIXTURE);
    }

    #[test]
    fn immutable_plainmark_v1_fixture_remains_decryptable() {
        assert_eq!(
            decrypt_text_value(V1_FIXTURE.into(), "archive-pass".into()).unwrap(),
            "# Legacy secret\nold note"
        );
    }

    #[test]
    fn unknown_versions_are_not_reinterpreted_as_v1() {
        let future = V1_FIXTURE.replacen("enc:v1:", "enc:v2:", 1);
        assert_eq!(
            decrypt_text_value(future, "archive-pass".into()).unwrap_err().code,
            "unsupportedEncryptedFormat"
        );
    }
}
