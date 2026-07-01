package keys_test

import (
	"os"
	"path/filepath"
	"testing"

	"docctl/internal/keys"
)

func TestGenerateKey(t *testing.T) {
	dir := t.TempDir()
	pin := []byte("test-pin-123")

	shortFP, err := keys.GenerateKey(dir, "testorg", "Test Signator", "Testroll", "test@example.com", pin, nil)
	if err != nil {
		t.Fatalf("GenerateKey returnerade fel: %v", err)
	}
	if len(shortFP) == 0 {
		t.Fatal("shortFP är tom")
	}

	expected := filepath.Join(dir, ".keys", "testorg_"+shortFP+".asc")
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf("förväntad nyckelfil saknas: %s", expected)
	}

	revoke := filepath.Join(dir, ".keys", "testorg_"+shortFP+"_REVOKE.asc")
	if _, err := os.Stat(revoke); err != nil {
		t.Fatalf("förväntad återkallningsfil saknas: %s", revoke)
	}
}

func TestListKeysForOrg(t *testing.T) {
	dir := t.TempDir()
	pin := []byte("pin")

	fp1, err := keys.GenerateKey(dir, "org1", "Alice", "Roll", "a@b.com", pin, nil)
	if err != nil {
		t.Fatalf("GenerateKey org1: %v", err)
	}
	_, err = keys.GenerateKey(dir, "org2", "Bob", "Roll", "b@b.com", pin, nil)
	if err != nil {
		t.Fatalf("GenerateKey org2: %v", err)
	}

	metas, err := keys.ListKeysForOrg(dir, "org1")
	if err != nil {
		t.Fatalf("ListKeysForOrg: %v", err)
	}
	if len(metas) != 1 {
		t.Fatalf("förväntade 1 nyckel för org1, fick %d", len(metas))
	}
	if metas[0].ShortFP != fp1 {
		t.Errorf("ShortFP mismatch: fick %s, vill ha %s", metas[0].ShortFP, fp1)
	}
	if metas[0].OrgID != "org1" {
		t.Errorf("OrgID mismatch: %s", metas[0].OrgID)
	}
}

func TestListAllKeys(t *testing.T) {
	dir := t.TempDir()
	pin := []byte("pin")

	for _, org := range []string{"a", "b", "c"} {
		if _, err := keys.GenerateKey(dir, org, "Namn", "Roll", org+"@test.com", pin, nil); err != nil {
			t.Fatalf("GenerateKey %s: %v", org, err)
		}
	}

	metas, err := keys.ListAllKeys(dir)
	if err != nil {
		t.Fatalf("ListAllKeys: %v", err)
	}
	if len(metas) != 3 {
		t.Fatalf("förväntade 3 nycklar, fick %d", len(metas))
	}
}

func TestLoadAndUnlock_CorrectPin(t *testing.T) {
	dir := t.TempDir()
	pin := []byte("rätt-pin")

	shortFP, err := keys.GenerateKey(dir, "myorg", "Sven Svensson", "Ordförande", "sven@test.com", pin, nil)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	keyPath := filepath.Join(dir, ".keys", "myorg_"+shortFP+".asc")
	entity, err := keys.LoadAndUnlock(keyPath, pin)
	if err != nil {
		t.Fatalf("LoadAndUnlock med rätt PIN misslyckades: %v", err)
	}
	if entity == nil {
		t.Fatal("entity är nil")
	}
	if entity.PrivateKey == nil {
		t.Fatal("PrivateKey är nil")
	}
	if entity.PrivateKey.Encrypted {
		t.Fatal("nyckeln är fortfarande krypterad efter upplåsning")
	}
}

func TestLoadAndUnlock_WrongPin(t *testing.T) {
	dir := t.TempDir()
	pin := []byte("rätt-pin")

	shortFP, err := keys.GenerateKey(dir, "myorg", "Anna Andersson", "Sekreterare", "anna@test.com", pin, nil)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	keyPath := filepath.Join(dir, ".keys", "myorg_"+shortFP+".asc")
	_, err = keys.LoadAndUnlock(keyPath, []byte("fel-pin"))
	if err == nil {
		t.Fatal("LoadAndUnlock borde misslyckas med fel PIN")
	}
}

func TestKeyMeta_Fields(t *testing.T) {
	dir := t.TempDir()
	pin := []byte("pin")

	_, err := keys.GenerateKey(dir, "testorg", "Karl Karlsson", "Kassör", "karl@test.com", pin, nil)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	metas, err := keys.ListKeysForOrg(dir, "testorg")
	if err != nil || len(metas) == 0 {
		t.Fatalf("ListKeysForOrg: %v (antal: %d)", err, len(metas))
	}

	m := metas[0]
	if m.Name != "Karl Karlsson" {
		t.Errorf("Name: fick %q", m.Name)
	}
	if m.Email != "karl@test.com" {
		t.Errorf("Email: fick %q", m.Email)
	}
	if m.Label != "Kassör" {
		t.Errorf("Label: fick %q", m.Label)
	}
	if m.Created.IsZero() {
		t.Error("Created är noll")
	}
}
