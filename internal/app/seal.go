package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
)

func DoSeal(kallorPath string, entity *openpgp.Entity, force bool) (err error) {
	kallorDir, _ := filepath.Abs(kallorPath)
	motesDir := filepath.Dir(kallorDir)
	arkivDir := filepath.Join(motesDir, "arkiv")
	manifestPath := filepath.Join(arkivDir, "ATTESTATION.md")
	sigPath := filepath.Join(arkivDir, "ATTESTATION.md.sig")
	pubKeyPath := filepath.Join(arkivDir, "PUBLIC_KEY.asc")

	if _, err := os.Stat(arkivDir); os.IsNotExist(err) {
		return fmt.Errorf("❌ Hittar inte arkiv-mappen! Kör 'build' först.")
	}

	if _, err := os.Stat(sigPath); err == nil && !force {
		return fmt.Errorf("❌ AVSLAGET: Arkivet är redan förseglat! Kör med --force.")
	}

	fmt.Println("🔒 Förseglar arkivet...")
	fmt.Println("   -> Exporterar signeringsnyckel (Public Key)...")

	if _, err := os.Stat(pubKeyPath); !os.IsNotExist(err) {
		err = os.Remove(pubKeyPath)
		if err != nil {
			return fmt.Errorf("could not remove public key: %w", err)
		}
	}

	{
		f, err := os.Create(pubKeyPath)
		if err != nil {
			return fmt.Errorf("could not create public key file: %w", err)
		}
		w, err := armor.Encode(f, "PGP PUBLIC KEY BLOCK", nil)
		if err != nil {
			_ = f.Close()
			return fmt.Errorf("could not create armor writer: %w", err)
		}
		if err := entity.Serialize(w); err != nil {
			_ = w.Close()
			_ = f.Close()
			return fmt.Errorf("could not export public key: %w", err)
		}
		_ = w.Close()
		_ = f.Close()
	}

	var files []string
	err = filepath.WalkDir(arkivDir, func(path string, d os.DirEntry, err error) error {
		if !d.IsDir() && d.Name() != "ATTESTATION.md" && d.Name() != "ATTESTATION.md.sig" && !isSystemFile(d.Name()) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(files)

	f, _ := os.Create(manifestPath)
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	if _, err := f.WriteString("# Arkivmanifest\n\n"); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}
	if _, err := f.WriteString(fmt.Sprintf("**Förseglat:** %s\n\n", time.Now().Format("2006-01-02 15:04:05"))); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	if _, err := f.WriteString("Undertecknad intygar härmed att nedanstående filer\nuthör det formellt justerade och godkända dokumentet.\n\n"); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	if _, err := f.WriteString("### Arkivinnehåll:\n"); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	for _, file := range files {
		relPath, err := filepath.Rel(arkivDir, file)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path")
		}
		hashStr, err := HashFile(file)
		if err != nil {
			return fmt.Errorf("failed to calculate hash")
		}

		line := fmt.Sprintf("- `%s` (SHA-256: `%s`)\n", relPath, hashStr)

		if _, err := f.WriteString(line); err != nil {
			return fmt.Errorf("failed to write to file")
		}
	}
	_ = f.Close()

	if _, err := os.Stat(sigPath); !os.IsNotExist(err) {
		err = os.Remove(sigPath)
		if err != nil {
			return fmt.Errorf("could not remove sig file: %w", err)
		}
	}

	{
		sigFile, err := os.Create(sigPath)
		if err != nil {
			return fmt.Errorf("could not create signature file: %w", err)
		}
		manifest, err := os.Open(manifestPath)
		if err != nil {
			_ = sigFile.Close()
			return fmt.Errorf("could not open manifest: %w", err)
		}
		if err := openpgp.ArmoredDetachSign(sigFile, entity, manifest, nil); err != nil {
			_ = manifest.Close()
			_ = sigFile.Close()
			return fmt.Errorf("could not sign manifest: %w", err)
		}
		_ = manifest.Close()
		_ = sigFile.Close()
	}

	// Koll om vi ska ZIPPA enligt inställningarna!
	var projRoot string
	curr := kallorDir
	for {
		if _, err := os.Stat(filepath.Join(curr, ".tooling")); !os.IsNotExist(err) {
			projRoot = curr
			break
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}

	if projRoot != "" {
		cfg, err := LoadConfig(projRoot)
		if err != nil {
			return err
		}
		if cfg.Settings.CreateZIP {
			fmt.Println("   -> Paketerar AIP (Archival Information Package)...")
			zipName := filepath.Base(motesDir) + "_Arkivpaket.zip"
			zipPath := filepath.Join(motesDir, zipName)
			err := CreateZipArchive(arkivDir, zipPath)
			if err != nil {
				fmt.Println("❌ Misslyckades med att skapa ZIP:", err)
			} else {
				fmt.Println("   -> ZIP sparad som:", zipName)
			}
		}
	}

	cfg, err := LoadConfig(projRoot)
	if err != nil {
		return err
	}

	// Only for OpenTimeStamps
	if cfg.Settings.UseOpenTimeStamps {
		// ============================================
		// OPENTIMESTAMPS (Tidsstämpling på Blockkedjan)
		// ============================================
		fmt.Println("   -> Söker efter OpenTimestamps för oantastligt tidsbevis...")

		// Kolla om 'ots' finns installerat på datorn
		_, errOTS := exec.LookPath("ots")
		if errOTS == nil {
			fmt.Println("   -> 'ots' hittades! Tidsstämplar manifestet mot Bitcoin-nätverket...")

			// Kör kommandot: ots stamp ATTESTATION.md.sig
			cmdOTS := exec.Command("ots", "stamp", sigPath)
			if err := cmdOTS.Run(); err != nil {
				fmt.Println("   ⚠️ Kunde inte nå OTS-servern just nu. Hoppar över tidsstämpel.")
			} else {
				fmt.Println("   ✅ Tidsstämpel skapad (.ots-fil sparad i arkivet).")
			}
		} else {
			fmt.Println("   💡 Tips: Installera 'opentimestamps-client' för att få gratis, kryptografiska tidsstämplar!")
		}
	}

	fmt.Println("✅ Arkivet är låst och GPG-signerat.")

	return nil
}
