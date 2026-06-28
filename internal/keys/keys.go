package keys

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

type KeyMeta struct {
	OrgID     string
	ShortFP   string
	Name      string
	Label     string
	Email     string
	Created   time.Time
	ExpiresAt *time.Time
	FilePath  string
}

func KeysDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".keys")
}

func shortFingerprint(fp []byte) string {
	full := strings.ToUpper(hex.EncodeToString(fp))
	if len(full) >= 8 {
		return full[len(full)-8:]
	}
	return full
}

func GenerateKey(projectRoot, orgID, name, label, email string, pin []byte, expiresAt *time.Time) (shortFP string, err error) {
	cfg := &packet.Config{Algorithm: packet.PubKeyAlgoEdDSA}
	if expiresAt != nil {
		secs := uint32(time.Until(*expiresAt).Seconds())
		cfg.KeyLifetimeSecs = secs
	}
	entity, err := openpgp.NewEntity(name, label, email, cfg)
	if err != nil {
		return "", fmt.Errorf("kunde inte generera nyckel: %w", err)
	}

	fp := entity.PrimaryKey.Fingerprint
	shortFP = shortFingerprint(fp)

	keysDir := KeysDir(projectRoot)
	if err := os.MkdirAll(keysDir, 0700); err != nil {
		return "", fmt.Errorf("kunde inte skapa nyckelkatalog: %w", err)
	}

	// Generera återkallningscertifikat INNAN privatnyckeln krypteras (Encrypt nollsätter PrivateKey.PrivateKey).
	if err := entity.RevokeKey(packet.NoReason, "", nil); err != nil {
		return "", fmt.Errorf("kunde inte generera återkallningscertifikat: %w", err)
	}
	revSig := entity.Revocations[0]
	entity.Revocations = nil // Rensa så certifikatet inte bäddas in i nyckelfilen

	revPath := filepath.Join(keysDir, fmt.Sprintf("%s_%s_REVOKE.asc", orgID, shortFP))
	fRev, err := os.OpenFile(revPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return "", fmt.Errorf("kunde inte skapa återkallningsfil: %w", err)
	}
	wRev, err := armor.Encode(fRev, "PGP REVOCATION CERTIFICATE", nil)
	if err != nil {
		_ = fRev.Close()
		return "", fmt.Errorf("kunde inte skapa armor-writer för återkallning: %w", err)
	}
	if err := revSig.Serialize(wRev); err != nil {
		_ = wRev.Close()
		_ = fRev.Close()
		return "", fmt.Errorf("kunde inte serialisera återkallningscertifikat: %w", err)
	}
	_ = wRev.Close()
	_ = fRev.Close()

	// Kryptera privatnyckel med PIN
	if err := entity.PrivateKey.Encrypt(pin); err != nil {
		return "", fmt.Errorf("kunde inte kryptera nyckel: %w", err)
	}
	for _, sub := range entity.Subkeys {
		if sub.PrivateKey != nil {
			if err := sub.PrivateKey.Encrypt(pin); err != nil {
				return "", fmt.Errorf("kunde inte kryptera undernyckel: %w", err)
			}
		}
	}

	keyPath := filepath.Join(keysDir, fmt.Sprintf("%s_%s.asc", orgID, shortFP))
	f, err := os.OpenFile(keyPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return "", fmt.Errorf("kunde inte skapa nyckelfil: %w", err)
	}
	defer f.Close()

	w, err := armor.Encode(f, "PGP PRIVATE KEY BLOCK", nil)
	if err != nil {
		return "", fmt.Errorf("kunde inte skapa armor-writer: %w", err)
	}
	if err := entity.SerializePrivateWithoutSigning(w, nil); err != nil {
		_ = w.Close()
		return "", fmt.Errorf("kunde inte serialisera nyckel: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	return shortFP, nil
}

func ListKeysForOrg(projectRoot, orgID string) ([]KeyMeta, error) {
	keysDir := KeysDir(projectRoot)
	pattern := filepath.Join(keysDir, orgID+"_*.asc")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var result []KeyMeta
	for _, path := range matches {
		meta, err := readKeyMeta(path)
		if err != nil {
			continue
		}
		result = append(result, meta)
	}
	return result, nil
}

func ListAllKeys(projectRoot string) ([]KeyMeta, error) {
	keysDir := KeysDir(projectRoot)
	pattern := filepath.Join(keysDir, "*.asc")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	var result []KeyMeta
	for _, path := range matches {
		meta, err := readKeyMeta(path)
		if err != nil {
			continue
		}
		result = append(result, meta)
	}
	return result, nil
}

func readKeyMeta(path string) (KeyMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return KeyMeta{}, err
	}
	defer f.Close()

	entities, err := openpgp.ReadArmoredKeyRing(f)
	if err != nil || len(entities) == 0 {
		return KeyMeta{}, fmt.Errorf("kunde inte läsa nyckel från %s", path)
	}

	e := entities[0]

	// Extrahera orgID från filnamnet: {orgID}_{fp}.asc
	base := strings.TrimSuffix(filepath.Base(path), ".asc")
	orgID := ""
	if idx := strings.LastIndex(base, "_"); idx > 0 {
		orgID = base[:idx]
	}

	meta := KeyMeta{
		OrgID:    orgID,
		ShortFP:  shortFingerprint(e.PrimaryKey.Fingerprint),
		Created:  e.PrimaryKey.CreationTime,
		FilePath: path,
	}

	for _, uid := range e.Identities {
		meta.Name = uid.UserId.Name
		meta.Label = uid.UserId.Comment
		meta.Email = uid.UserId.Email
		if uid.SelfSignature != nil && uid.SelfSignature.KeyLifetimeSecs != nil && *uid.SelfSignature.KeyLifetimeSecs > 0 {
			t := e.PrimaryKey.CreationTime.Add(time.Duration(*uid.SelfSignature.KeyLifetimeSecs) * time.Second)
			meta.ExpiresAt = &t
		}
		break
	}

	return meta, nil
}

func LoadAndUnlock(filePath string, pin []byte) (*openpgp.Entity, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("kunde inte öppna nyckelfil: %w", err)
	}
	defer f.Close()

	entities, err := openpgp.ReadArmoredKeyRing(f)
	if err != nil || len(entities) == 0 {
		return nil, fmt.Errorf("kunde inte läsa nyckel")
	}

	e := entities[0]

	if e.PrivateKey != nil && e.PrivateKey.Encrypted {
		if err := e.PrivateKey.Decrypt(pin); err != nil {
			return nil, fmt.Errorf("fel PIN-kod")
		}
	}
	for _, sub := range e.Subkeys {
		if sub.PrivateKey != nil && sub.PrivateKey.Encrypted {
			_ = sub.PrivateKey.Decrypt(pin)
		}
	}

	return e, nil
}
