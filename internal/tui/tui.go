package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"docctl/internal/app"
	"docctl/internal/assets"
	"docctl/internal/keys"

	"github.com/ProtonMail/go-crypto/openpgp"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

type AppContext struct {
	Cwd string
	Cfg *app.Config
}

type appState int

const (
	stateMainMenu appState = iota
	stateSettingsMenu
	stateSettingsCreateOrg
	stateSettingsChooseOrgToEdit
	stateSettingsEditOrgDetails
	stateEditZip
	stateEditOts
	stateChooseOrgForBuild
	stateChooseOrgForSeal
	stateChooseDocumentToSeal
	stateSelectKeyForSeal
	stateEnterPinForSeal
	stateKeyManagementMenu
	stateGenerateKey
	stateListKeys
	stateChooseOrgForDoc
	stateCreateOrganization
	stateFirstRun
	stateCreateBody
	stateChooseBodyForm
	stateChooseDocumentTypeForm
	stateAskDateForm
	stateChooseGoverningDocumentTypeForm
	stateCreateNewGoverningDocumentTypeForm
	stateAskGoverningDocumentNameForm
	stateChooseOtherDocumentType
	stateChooseOtherDocumentCategoryNameForm
	stateCreateOtherDocumentTypeForm
	stateChooseDocumentToBuild
	stateAskIfUserWantToNukeSealedDocument
	stateAskIfUserWantToNukeSealedDocumentForSealedDocument
	stateConfirmTemplateUpdate
	stateExportPublicKey
	stateGenerateKeyContract
	stateKeyActionResult
	stateChooseSourceFormat
	stateChooseFODTTemplate
)

type mainModel struct {
	state appState
	cwd   string
	cfg   *app.Config

	selectedOrg             app.Association
	selectedDocumentType    DocumentType
	selectedBody            string
	date                    string
	force                   bool
	selectedDocumentToBuild string
	selectedDocumentToSeal  string

	selectedGoverningDocType string
	selectedGoverningDocName string
	grundaktCategory         string

	selectedSourceFormat SourceFormat
	selectedFODTTemplate string

	editOrgName   *string
	editOrgNumber *string

	// Nyckelhantering
	availableKeys   []keys.KeyMeta
	selectedKeyMeta keys.KeyMeta
	unlockedEntity  *openpgp.Entity

	activeForm *huh.Form

	// Formulär
	mainMenu     *huh.Form
	settingsMenu *huh.Form

	errMsg    string
	resultMsg string
}

func (m mainModel) Init() tea.Cmd {
	return nil
}

// stepActiveForm uppdaterar activeForm och returnerar den resulterande cmd.
func (m mainModel) stepActiveForm(msg tea.Msg) (mainModel, tea.Cmd) {
	form, cmd := m.activeForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.activeForm = f
	}
	return m, cmd
}

// goToMainMenu är ett hjälpmedel för att återgå till huvudmenyn.
func (m mainModel) goToMainMenu() (mainModel, tea.Cmd) {
	m.state = stateMainMenu
	m.mainMenu = createMainMenuForm()
	return m, m.mainMenu.Init()
}

// goToSettingsMenu är ett hjälpmedel för att återgå till inställningsmenyn.
func (m mainModel) goToSettingsMenu() (mainModel, tea.Cmd) {
	m.state = stateSettingsMenu
	m.settingsMenu = createSettingsMenuForm()
	return m, m.settingsMenu.Init()
}

// goToKeyManagementMenu är ett hjälpmedel för att återgå till nyckelhanteringsmenyn.
func (m mainModel) goToKeyManagementMenu() (mainModel, tea.Cmd) {
	m.state = stateKeyManagementMenu
	m.activeForm = createKeyManagementMenuForm()
	return m, m.activeForm.Init()
}

// ============================================
// UPPDATERINGSMETODER PER STATE
// ============================================

func (m mainModel) updateMainMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	form, cmd := m.mainMenu.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.mainMenu = f
	}
	cmds = append(cmds, cmd)

	if m.mainMenu.State == huh.StateCompleted {
		action := m.mainMenu.GetString("action")
		m.mainMenu = createMainMenuForm()

		switch action {
		case "settings":
			m.state = stateSettingsMenu
			m.settingsMenu = createSettingsMenuForm()
			cmds = append(cmds, m.settingsMenu.Init())

		case "init":
			*m.cfg, _ = app.LoadConfig(m.cwd)
			m.state = stateChooseOrgForDoc
			m.activeForm = createChooseOrgForm(m.cfg.Foreningar, "Välj förening", true)
			cmds = append(cmds, m.activeForm.Init())

		case "build":
			m.state = stateChooseOrgForBuild
			m.activeForm = createChooseOrgForm(m.cfg.Foreningar, "Välj förening", false)
			cmds = append(cmds, m.activeForm.Init())

		case "seal":
			m.state = stateChooseOrgForSeal
			m.activeForm = createChooseOrgForm(m.cfg.Foreningar, "Välj förening", false)
			cmds = append(cmds, m.activeForm.Init())

		case "exit":
			return m, tea.Quit
		}
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateSettingsMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	form, cmd := m.settingsMenu.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.settingsMenu = f
	}
	cmds = append(cmds, cmd)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEsc {
			m.state = stateMainMenu
			m.settingsMenu = createSettingsMenuForm()
			cmds = append(cmds, m.mainMenu.Init())
			return m, tea.Batch(cmds...)
		}
	}

	if m.settingsMenu.State == huh.StateCompleted {
		action := m.settingsMenu.GetString("action")
		m.settingsMenu = createSettingsMenuForm()

		switch action {
		case "ny_org":
			m.activeForm = askForNewOrgDetailsForm()
			m.state = stateSettingsCreateOrg
			cmds = append(cmds, m.activeForm.Init())
		case "redigera_org":
			m.activeForm = createChooseOrgForm(m.cfg.Foreningar, "Välj förening att redigera", false)
			m.state = stateSettingsChooseOrgToEdit
			cmds = append(cmds, m.activeForm.Init())
		case "toggle_zip":
			m.activeForm = createConfirmForm("zip", "Skapa automatiskt ZIP-arkiv?", "Paketerar alla filer i en zip-fil när bygget är klart.", m.cfg.Settings.CreateZIP)
			m.state = stateEditZip
			cmds = append(cmds, m.activeForm.Init())
		case "toggle_ots":
			m.activeForm = createConfirmForm("ots", "Använd OpenTimestamps?", "Tidsstämplar sealed filer via blockkedjieteknik.", m.cfg.Settings.UseOpenTimeStamps)
			m.state = stateEditOts
			cmds = append(cmds, m.activeForm.Init())
		case "update_templates":
			m.activeForm = askConfirmTemplateUpdateForm()
			m.state = stateConfirmTemplateUpdate
			cmds = append(cmds, m.activeForm.Init())
		case "key_management":
			m.state = stateKeyManagementMenu
			m.activeForm = createKeyManagementMenuForm()
			cmds = append(cmds, m.activeForm.Init())
		case "back":
			m.state = stateMainMenu
			cmds = append(cmds, m.mainMenu.Init())
		}
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateSettingsCreateOrg(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEsc {
			var settingsCmd tea.Cmd
			m, settingsCmd = m.goToSettingsMenu()
			return m, tea.Batch(append(cmds, settingsCmd)...)
		}
	}

	if m.activeForm.State == huh.StateCompleted {
		newOrgId := m.activeForm.GetString("org_id")
		newOrgName := m.activeForm.GetString("org_name")
		newOrgNumber := m.activeForm.GetString("org_number")

		if newOrgId != "" {
			if err := app.CreateOrganization(m.cwd, newOrgId, newOrgName, newOrgNumber); err == nil {
				*m.cfg, _ = app.LoadConfig(m.cwd)
			}
		}

		var settingsCmd tea.Cmd
		m, settingsCmd = m.goToSettingsMenu()
		cmds = append(cmds, settingsCmd)
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateSettingsChooseOrgToEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEsc {
			var settingsCmd tea.Cmd
			m, settingsCmd = m.goToSettingsMenu()
			return m, tea.Batch(append(cmds, settingsCmd)...)
		}
	}

	if m.activeForm.State == huh.StateCompleted {
		selectedOrgId := m.activeForm.GetString("selected_org")
		switch selectedOrgId {
		case "back":
			var settingsCmd tea.Cmd
			m, settingsCmd = m.goToSettingsMenu()
			cmds = append(cmds, settingsCmd)
		default:
			m.selectedOrg = m.cfg.Foreningar[selectedOrgId]
			nameVal := m.selectedOrg.Name
			numberVal := m.selectedOrg.OrgNummer
			m.editOrgName = &nameVal
			m.editOrgNumber = &numberVal

			m.activeForm = editOrgDetailsForm(m.editOrgName, m.editOrgNumber)
			m.state = stateSettingsEditOrgDetails
			cmds = append(cmds, m.activeForm.Init())
		}
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateSettingsEditOrgDetails(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEsc {
			m.state = stateSettingsChooseOrgToEdit
			m.activeForm = createChooseOrgForm(m.cfg.Foreningar, "Välj förening att redigera", false)
			cmds = append(cmds, m.activeForm.Init())
			return m, tea.Batch(cmds...)
		}
	}

	if m.activeForm.State == huh.StateCompleted {
		m.selectedOrg.Name = *m.editOrgName
		m.selectedOrg.OrgNummer = *m.editOrgNumber
		m.cfg.Foreningar[m.selectedOrg.Id] = m.selectedOrg
		_ = app.SaveConfig(m.cwd, *m.cfg)

		var settingsCmd tea.Cmd
		m, settingsCmd = m.goToSettingsMenu()
		cmds = append(cmds, settingsCmd)
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateEditZip(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		m.cfg.Settings.CreateZIP = m.activeForm.GetBool("zip")
		if err := app.SaveConfig(m.cwd, *m.cfg); err != nil {
			fmt.Println("Error saving config:", err)
		}
		var settingsCmd tea.Cmd
		m, settingsCmd = m.goToSettingsMenu()
		cmds = append(cmds, settingsCmd)
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateEditOts(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		m.cfg.Settings.UseOpenTimeStamps = m.activeForm.GetBool("ots")
		if err := app.SaveConfig(m.cwd, *m.cfg); err != nil {
			fmt.Println("Error saving config:", err)
		}
		var settingsCmd tea.Cmd
		m, settingsCmd = m.goToSettingsMenu()
		cmds = append(cmds, settingsCmd)
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateConfirmTemplateUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		if m.activeForm.GetBool("confirm") {
			_ = app.UpdateTemplates(m.cwd, assets.Files)
		}
		var settingsCmd tea.Cmd
		m, settingsCmd = m.goToSettingsMenu()
		cmds = append(cmds, settingsCmd)
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateChooseOrgForBuild(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		*m.cfg, _ = app.LoadConfig(m.cwd)
		m.selectedOrg = m.cfg.Foreningar[m.activeForm.GetString("selected_org")]
		m.state = stateChooseDocumentToBuild
		m.activeForm = listAllDocumentsFromAssociation(m.cwd, m.selectedOrg)
		cmds = append(cmds, m.activeForm.Init())
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateChooseDocumentToBuild(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.force {
		app.DoBuild(filepath.Join(m.selectedOrg.Id, m.selectedDocumentToBuild, "källor"), m.force)
		m.force = false
		var mainCmd tea.Cmd
		m, mainCmd = m.goToMainMenu()
		return m, tea.Batch(append(cmds, mainCmd)...)
	}

	if m.activeForm.State == huh.StateCompleted {
		m.selectedDocumentToBuild = m.activeForm.GetString("document")
		switch m.selectedDocumentToBuild {
		case "back":
			var mainCmd tea.Cmd
			m, mainCmd = m.goToMainMenu()
			cmds = append(cmds, mainCmd)
		default:
			sigPath := filepath.Join(m.cwd, m.selectedOrg.Id, m.selectedDocumentToBuild, "arkiv", "ATTESTATION.md.sig")
			if _, err := os.Stat(sigPath); err == nil {
				m.state = stateAskIfUserWantToNukeSealedDocument
				m.activeForm = askGenericYesOrNowForm("⚠️ Arkivet/Avtalet är förseglat!", " Byggs det om raderas signaturen. Fortsätta?", "action")
				cmds = append(cmds, m.activeForm.Init())
			} else {
				app.DoBuild(filepath.Join(m.selectedOrg.Id, m.selectedDocumentToBuild, "källor"), m.force)
				var mainCmd tea.Cmd
				m, mainCmd = m.goToMainMenu()
				cmds = append(cmds, mainCmd)
			}
		}
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateAskNukeSealedDocument(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		m.force = m.activeForm.GetBool("action")
		if m.force {
			m.state = stateChooseDocumentToBuild
			m.activeForm = listAllDocumentsFromAssociation(m.cwd, m.selectedOrg)
			cmds = append(cmds, m.activeForm.Init())
		} else {
			var mainCmd tea.Cmd
			m, mainCmd = m.goToMainMenu()
			cmds = append(cmds, mainCmd)
		}
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateChooseOrgForSeal(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		*m.cfg, _ = app.LoadConfig(m.cwd)
		m.selectedOrg = m.cfg.Foreningar[m.activeForm.GetString("selected_org")]
		m.state = stateChooseDocumentToSeal
		m.activeForm = listAllBuildDocumentsFromAssociation(m.cwd, m.selectedOrg)
		cmds = append(cmds, m.activeForm.Init())
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateChooseDocumentToSeal(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.force {
		var keyCmd tea.Cmd
		m, keyCmd = m.goToKeySelect()
		cmds = append(cmds, keyCmd)
		return m, tea.Batch(cmds...)
	}

	if m.activeForm.State == huh.StateCompleted {
		m.selectedDocumentToSeal = m.activeForm.GetString("document")
		switch m.selectedDocumentToSeal {
		case "back":
			var mainCmd tea.Cmd
			m, mainCmd = m.goToMainMenu()
			cmds = append(cmds, mainCmd)
		default:
			sigPath := filepath.Join(m.cwd, m.selectedOrg.Id, m.selectedDocumentToSeal, "arkiv", "ATTESTATION.md.sig")
			if _, err := os.Stat(sigPath); err == nil {
				m.state = stateAskIfUserWantToNukeSealedDocumentForSealedDocument
				m.activeForm = askGenericYesOrNowForm("⚠️ Arkivet/Avtalet är förseglat!", " Seals det om raderas signaturen. Fortsätta?", "action")
				cmds = append(cmds, m.activeForm.Init())
			} else {
				var keyCmd tea.Cmd
				m, keyCmd = m.goToKeySelect()
				cmds = append(cmds, keyCmd)
			}
		}
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateAskNukeSealedForSeal(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		m.force = m.activeForm.GetBool("action")
		if m.force {
			m.state = stateChooseDocumentToSeal
			m.activeForm = listAllBuildDocumentsFromAssociation(m.cwd, m.selectedOrg)
			cmds = append(cmds, m.activeForm.Init())
		} else {
			var mainCmd tea.Cmd
			m, mainCmd = m.goToMainMenu()
			cmds = append(cmds, mainCmd)
		}
	}
	return m, tea.Batch(cmds...)
}

// goToKeySelect laddar tillgängliga nycklar och går till rätt state.
func (m mainModel) goToKeySelect() (mainModel, tea.Cmd) {
	orgKeys, err := keys.ListKeysForOrg(m.cwd, m.selectedOrg.Id)
	if err != nil || len(orgKeys) == 0 {
		fmt.Println("❌ Inga signeringsnycklar hittades för", m.selectedOrg.Id, "– gå till Inställningar → Nyckelhantering för att skapa en.")
		return m.goToMainMenu()
	}

	m.availableKeys = orgKeys

	if len(orgKeys) == 1 {
		// Bara en nyckel – välj direkt och gå till PIN-input
		m.selectedKeyMeta = orgKeys[0]
		m.state = stateEnterPinForSeal
		m.activeForm = createPinInputForm()
		return m, m.activeForm.Init()
	}

	// Flera nycklar – visa val
	m.state = stateSelectKeyForSeal
	m.activeForm = createKeySelectionForm(orgKeys)
	return m, m.activeForm.Init()
}

func (m mainModel) updateSelectKeyForSeal(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		selectedFP := m.activeForm.GetString("key")
		for _, k := range m.availableKeys {
			if k.ShortFP == selectedFP {
				m.selectedKeyMeta = k
				break
			}
		}
		m.state = stateEnterPinForSeal
		m.activeForm = createPinInputForm()
		cmds = append(cmds, m.activeForm.Init())
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateEnterPinForSeal(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		pin := []byte(m.activeForm.GetString("pin"))
		entity, err := keys.LoadAndUnlock(m.selectedKeyMeta.FilePath, pin)
		if err != nil {
			fmt.Println("❌", err, "– försök igen.")
			m.activeForm = createPinInputForm()
			cmds = append(cmds, m.activeForm.Init())
			return m, tea.Batch(cmds...)
		}

		if err := app.DoSeal(filepath.Join(m.selectedOrg.Id, m.selectedDocumentToSeal, "källor"), entity, m.force); err != nil {
			fmt.Println(err)
		}
		m.force = false
		m.unlockedEntity = nil
		var mainCmd tea.Cmd
		m, mainCmd = m.goToMainMenu()
		cmds = append(cmds, mainCmd)
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateKeyManagementMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEsc {
			var settingsCmd tea.Cmd
			m, settingsCmd = m.goToSettingsMenu()
			return m, tea.Batch(append(cmds, settingsCmd)...)
		}
	}

	if m.activeForm.State == huh.StateCompleted {
		switch m.activeForm.GetString("action") {
		case "generate":
			m.state = stateGenerateKey
			m.activeForm = createGenerateKeyForm(m.cfg.Foreningar)
			cmds = append(cmds, m.activeForm.Init())
		case "list":
			allKeys, err := keys.ListAllKeys(m.cwd)
			if err != nil || len(allKeys) == 0 {
				fmt.Println("Inga nycklar hittades i", m.cwd+"/.keys/")
				m.activeForm = createKeyManagementMenuForm()
				cmds = append(cmds, m.activeForm.Init())
			} else {
				m.state = stateListKeys
				m.activeForm = createKeyListForm(allKeys)
				cmds = append(cmds, m.activeForm.Init())
			}
		case "export_key":
			allKeys, err := keys.ListAllKeys(m.cwd)
			if err != nil || len(allKeys) == 0 {
				fmt.Println("Inga nycklar hittades i", m.cwd+"/.keys/")
				m.activeForm = createKeyManagementMenuForm()
				cmds = append(cmds, m.activeForm.Init())
			} else {
				m.availableKeys = allKeys
				m.state = stateExportPublicKey
				m.activeForm = createKeySelectionForActionForm(allKeys, "Välj nyckel att exportera")
				cmds = append(cmds, m.activeForm.Init())
			}
		case "key_contract":
			allKeys, err := keys.ListAllKeys(m.cwd)
			if err != nil || len(allKeys) == 0 {
				fmt.Println("Inga nycklar hittades i", m.cwd+"/.keys/")
				m.activeForm = createKeyManagementMenuForm()
				cmds = append(cmds, m.activeForm.Init())
			} else {
				m.availableKeys = allKeys
				m.state = stateGenerateKeyContract
				m.activeForm = createKeyContractForm(allKeys)
				cmds = append(cmds, m.activeForm.Init())
			}
		case "back":
			var settingsCmd tea.Cmd
			m, settingsCmd = m.goToSettingsMenu()
			cmds = append(cmds, settingsCmd)
		}
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateListKeys(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEsc {
			m.state = stateKeyManagementMenu
			m.activeForm = createKeyManagementMenuForm()
			cmds = append(cmds, m.activeForm.Init())
			return m, tea.Batch(cmds...)
		}
	}

	if m.activeForm.State == huh.StateCompleted {
		m.state = stateKeyManagementMenu
		m.activeForm = createKeyManagementMenuForm()
		cmds = append(cmds, m.activeForm.Init())
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateGenerateKey(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		orgID := m.activeForm.GetString("org")
		name := m.activeForm.GetString("name")
		label := m.activeForm.GetString("label")
		email := m.activeForm.GetString("email")
		expiry := m.activeForm.GetString("expiry")
		pin := m.activeForm.GetString("pin")
		pinConfirm := m.activeForm.GetString("pin_confirm")

		if pin != pinConfirm {
			fmt.Println("❌ PIN-koderna matchar inte – försök igen.")
			m.activeForm = createGenerateKeyForm(m.cfg.Foreningar)
			cmds = append(cmds, m.activeForm.Init())
			return m, tea.Batch(cmds...)
		}

		var expiresAt *time.Time
		switch expiry {
		case "1y":
			t := time.Now().AddDate(1, 0, 0)
			expiresAt = &t
		case "2y":
			t := time.Now().AddDate(2, 0, 0)
			expiresAt = &t
		case "3y":
			t := time.Now().AddDate(3, 0, 0)
			expiresAt = &t
		}

		shortFP, err := keys.GenerateKey(m.cwd, orgID, name, label, email, []byte(pin), expiresAt)
		if err != nil {
			fmt.Println("❌ Kunde inte generera nyckel:", err)
		} else {
			fmt.Printf("✅ Nyckel skapad! Fingeravtryck: %s\n", shortFP)
		}

		m.state = stateKeyManagementMenu
		m.activeForm = createKeyManagementMenuForm()
		cmds = append(cmds, m.activeForm.Init())
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateExportPublicKey(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEsc {
			var keyCmd tea.Cmd
			m, keyCmd = m.goToKeyManagementMenu()
			return m, tea.Batch(append(cmds, keyCmd)...)
		}
	}

	if m.activeForm.State == huh.StateCompleted {
		shortFP := m.activeForm.GetString("key")
		var meta keys.KeyMeta
		for _, k := range m.availableKeys {
			if k.ShortFP == shortFP {
				meta = k
				break
			}
		}
		outputPath := filepath.Join(m.cwd, ".keys",
			fmt.Sprintf("%s_%s_PUBLIC_KEY.asc", meta.OrgID, meta.ShortFP))
		if err := app.ExportPublicKey(meta.FilePath, outputPath); err != nil {
			m.resultMsg = fmt.Sprintf("❌ Kunde inte exportera nyckel:\n%v", err)
		} else {
			m.resultMsg = fmt.Sprintf("Publik nyckel exporterad till:\n%s", outputPath)
		}
		m.state = stateKeyActionResult
		m.activeForm = createKeyActionResultForm(m.resultMsg)
		cmds = append(cmds, m.activeForm.Init())
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateGenerateKeyContract(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEsc {
			var keyCmd tea.Cmd
			m, keyCmd = m.goToKeyManagementMenu()
			return m, tea.Batch(append(cmds, keyCmd)...)
		}
	}

	if m.activeForm.State == huh.StateCompleted {
		shortFP := m.activeForm.GetString("key")
		keyDir := m.activeForm.GetString("key_dir")
		if keyDir == "" {
			keyDir = "[NYCKELKATALOG URL]"
		}
		var meta keys.KeyMeta
		for _, k := range m.availableKeys {
			if k.ShortFP == shortFP {
				meta = k
				break
			}
		}
		org := m.cfg.Foreningar[meta.OrgID]
		outputPath, err := app.GenerateKeyContract(m.cwd, meta, org.Name, org.OrgNummer, keyDir)
		if err != nil {
			m.resultMsg = fmt.Sprintf("❌ Kunde inte generera nyckelkontrakt:\n%v", err)
		} else {
			m.resultMsg = fmt.Sprintf("Nyckelkontrakt sparat till:\n%s", outputPath)
		}
		m.state = stateKeyActionResult
		m.activeForm = createKeyActionResultForm(m.resultMsg)
		cmds = append(cmds, m.activeForm.Init())
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateKeyActionResult(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		var keyCmd tea.Cmd
		m, keyCmd = m.goToKeyManagementMenu()
		cmds = append(cmds, keyCmd)
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateChooseOrgForDoc(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEsc {
			var mainCmd tea.Cmd
			m, mainCmd = m.goToMainMenu()
			return m, tea.Batch(append(cmds, mainCmd)...)
		}
	}

	if m.activeForm.State == huh.StateCompleted {
		switch m.activeForm.GetString("selected_org") {
		case "create_new":
			m.activeForm = askForNewOrgDetailsForm()
			m.state = stateCreateOrganization
			cmds = append(cmds, m.activeForm.Init())
		case "back":
			var mainCmd tea.Cmd
			m, mainCmd = m.goToMainMenu()
			cmds = append(cmds, mainCmd)
		default:
			m.selectedOrg = m.cfg.Foreningar[m.activeForm.GetString("selected_org")]
			m.state = stateChooseDocumentTypeForm
			m.activeForm = chooseDocumentTypeForm()
			cmds = append(cmds, m.activeForm.Init())
		}
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateCreateOrganization(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		newOrg := app.Association{
			Id:        m.activeForm.GetString("org_id"),
			Name:      m.activeForm.GetString("org_name"),
			OrgNummer: m.activeForm.GetString("org_number"),
		}
		if err := app.CreateOrganization(m.cwd, newOrg.Id, newOrg.Name, newOrg.OrgNummer); err != nil {
			return nil, nil
		}
		*m.cfg, _ = app.LoadConfig(m.cwd)
		m.state = stateChooseOrgForDoc
		m.activeForm = createChooseOrgForm(m.cfg.Foreningar, "Välj förening", true)
		cmds = append(cmds, m.activeForm.Init())
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateFirstRun(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		if m.activeForm.GetBool("action") {
			if err := app.FirstTimeRun(filepath.Join(m.cwd, ".tooling"), m.cwd, assets.Files); err != nil {
				m.errMsg = fmt.Sprintf("Kunde inte skapa workspace: %v", err)
				return m, tea.Quit
			}
		} else {
			return m, tea.Quit
		}
		*m.cfg, _ = app.LoadConfig(m.cwd)
		var mainCmd tea.Cmd
		m, mainCmd = m.goToMainMenu()
		cmds = append(cmds, mainCmd)
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateChooseDocumentType(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEsc {
			*m.cfg, _ = app.LoadConfig(m.cwd)
			m.state = stateChooseOrgForDoc
			m.activeForm = createChooseOrgForm(m.cfg.Foreningar, "Välj förening", true)
			cmds = append(cmds, m.activeForm.Init())
			return m, tea.Batch(cmds...)
		}
	}

	if m.activeForm.State == huh.StateCompleted {
		m.selectedDocumentType = DocumentType(m.activeForm.GetString("document_type"))
		switch m.selectedDocumentType {
		case DocumentTypeProtokoll:
			m.activeForm = chooseBodyForm(m.selectedOrg)
			m.state = stateChooseBodyForm
			cmds = append(cmds, m.activeForm.Init())
		case DocumentTypeStyrdokument:
			m.state = stateChooseGoverningDocumentTypeForm
			m.activeForm = chooseSubcategoryForm(m.cwd, m.selectedOrg, []string{"Grundakter", "Styrdokument"}, "Vilken typ av styrdokument?")
			cmds = append(cmds, m.activeForm.Init())
		case DocumentTypeOther:
			m.state = stateChooseOtherDocumentType
			m.activeForm = chooseSubcategoryForm(m.cwd, m.selectedOrg, []string{"Grundakter"}, "Vilken typ av Grundakt?", "Styrdokument")
			cmds = append(cmds, m.activeForm.Init())
		}
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateChooseBody(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEsc {
			m.state = stateChooseDocumentTypeForm
			m.activeForm = chooseDocumentTypeForm()
			cmds = append(cmds, m.activeForm.Init())
			return m, tea.Batch(cmds...)
		}
	}

	if m.activeForm.State == huh.StateCompleted {
		switch m.activeForm.GetString("selected_body") {
		case "create_new":
			m.state = stateCreateBody
			m.activeForm = genericInputForm("Organets namn (t.ex. utbildningsutskott)", "name")
			cmds = append(cmds, m.activeForm.Init())
		default:
			m.selectedBody = m.activeForm.GetString("selected_body")
			m.state = stateAskDateForm
			m.activeForm = genericInputForm("Datum (ÅÅÅÅ-MM-DD):", "date")
			cmds = append(cmds, m.activeForm.Init())
		}
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateCreateBody(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		body := m.activeForm.GetString("name")
		*m.cfg, _ = app.LoadConfig(m.cwd)
		m.selectedOrg.Body = append(m.selectedOrg.Body, body)
		m.cfg.Foreningar[m.selectedOrg.Id] = m.selectedOrg
		_ = app.SaveConfig(m.cwd, *m.cfg)

		m.state = stateChooseBodyForm
		m.activeForm = chooseBodyForm(m.selectedOrg)
		cmds = append(cmds, m.activeForm.Init())
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateAskDate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		m.date = m.activeForm.GetString("date")
		m.state = stateChooseSourceFormat
		m.activeForm = chooseSourceFormatForm()
		cmds = append(cmds, m.activeForm.Init())
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateChooseGoverningDocumentType(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		switch m.activeForm.GetString("selected_document_type") {
		case "create_new":
			m.state = stateCreateNewGoverningDocumentTypeForm
			m.activeForm = genericInputForm("Kategorins namn (t.ex. Policy, Reglemente, Stadgar):", "name")
			cmds = append(cmds, m.activeForm.Init())
		default:
			m.selectedGoverningDocType = m.activeForm.GetString("selected_document_type")
			m.state = stateAskGoverningDocumentNameForm
			m.activeForm = genericInputForm("Dokumentets/Filens namn (t.ex. IT-policy):", "name")
			cmds = append(cmds, m.activeForm.Init())
		}
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateCreateNewGoverningDocumentType(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		name := m.activeForm.GetString("name")
		if err := os.MkdirAll(filepath.Join(m.cwd, m.selectedOrg.Id, "Grundakter", "Styrdokument", name), os.ModePerm); err != nil {
			m.errMsg = fmt.Sprintf("Kunde inte skapa mapp: %v", err)
			return m, tea.Quit
		}
		m.state = stateChooseGoverningDocumentTypeForm
		m.activeForm = chooseSubcategoryForm(m.cwd, m.selectedOrg, []string{"Grundakter", "Styrdokument"}, "Vilken typ av styrdokument?")
		cmds = append(cmds, m.activeForm.Init())
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateAskGoverningDocumentName(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		m.selectedGoverningDocName = m.activeForm.GetString("name")
		m.state = stateChooseSourceFormat
		m.activeForm = chooseSourceFormatForm()
		cmds = append(cmds, m.activeForm.Init())
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateChooseOtherDocumentType(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		switch m.activeForm.GetString("selected_document_type") {
		case "create_new":
			m.state = stateChooseOtherDocumentCategoryNameForm
			m.activeForm = genericInputForm("Kategorins namn (t.ex. Avtal, Motioner, Propositioner)", "name")
			cmds = append(cmds, m.activeForm.Init())
		default:
			m.grundaktCategory = m.activeForm.GetString("selected_document_type")
			m.state = stateCreateOtherDocumentTypeForm
			m.activeForm = genericInputForm("Dokumentets/Filens namn (t.ex Motion angående xx)", "name")
			cmds = append(cmds, m.activeForm.Init())
		}
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateChooseOtherDocumentCategoryName(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		name := m.activeForm.GetString("name")
		_ = os.MkdirAll(filepath.Join(m.cwd, m.selectedOrg.Id, "Grundakter", m.grundaktCategory, name), os.ModePerm)
		m.state = stateChooseOtherDocumentType
		m.activeForm = chooseSubcategoryForm(m.cwd, m.selectedOrg, []string{"Grundakter"}, "Vilken typ av Grundakt?", "Styrdokument")
		cmds = append(cmds, m.activeForm.Init())
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateCreateOtherDocumentType(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		m.selectedGoverningDocName = m.activeForm.GetString("name")
		m.state = stateChooseSourceFormat
		m.activeForm = chooseSourceFormatForm()
		cmds = append(cmds, m.activeForm.Init())
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateChooseSourceFormat(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		m.selectedSourceFormat = SourceFormat(m.activeForm.GetString("source_format"))

		if m.selectedSourceFormat == SourceFormatMarkdown {
			// Skapa Markdown-dokument som tidigare
			switch m.selectedDocumentType {
			case DocumentTypeProtokoll:
				_ = app.CreateProtokoll(m.cwd, m.selectedOrg.Id, m.selectedBody, m.date)
			case DocumentTypeStyrdokument:
				_ = app.CreateGuidanceDocuments(m.cwd, m.selectedOrg.Id, m.selectedGoverningDocType, m.selectedGoverningDocName)
			case DocumentTypeOther:
				_ = app.CreateOtherGoverningDocuments(m.cwd, m.selectedOrg.Id, m.grundaktCategory, m.selectedGoverningDocName)
			}
			var mainCmd tea.Cmd
			m, mainCmd = m.goToMainMenu()
			cmds = append(cmds, mainCmd)
		} else {
			// FODT: visa mallväljare
			templates := app.ListFODTTemplates(m.cwd)
			m.state = stateChooseFODTTemplate
			m.activeForm = chooseFODTTemplateForm(templates)
			cmds = append(cmds, m.activeForm.Init())
		}
	}
	return m, tea.Batch(cmds...)
}

func (m mainModel) updateChooseFODTTemplate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m, cmd := m.stepActiveForm(msg)
	cmds = append(cmds, cmd)

	if m.activeForm.State == huh.StateCompleted {
		m.selectedFODTTemplate = m.activeForm.GetString("fodt_template")

		var källorDir, docName string
		switch m.selectedDocumentType {
		case DocumentTypeProtokoll:
			if len(m.date) >= 4 {
				year := m.date[:4]
				källorDir = filepath.Join(m.cwd, m.selectedOrg.Id, "Årsakter", year, m.selectedBody, m.date, "källor")
			}
			docName = "protokoll"
		case DocumentTypeStyrdokument:
			safeType := strings.ReplaceAll(m.selectedGoverningDocType, " ", "-")
			safeName := strings.ReplaceAll(m.selectedGoverningDocName, " ", "-")
			källorDir = filepath.Join(m.cwd, m.selectedOrg.Id, "Grundakter", "Styrdokument", safeType, safeName, "källor")
			docName = "document"
		case DocumentTypeOther:
			safeCategory := strings.ReplaceAll(m.grundaktCategory, " ", "-")
			safeName := strings.ReplaceAll(m.selectedGoverningDocName, " ", "-")
			källorDir = filepath.Join(m.cwd, m.selectedOrg.Id, "Grundakter", safeCategory, safeName, "källor")
			docName = "document"
		}

		if källorDir != "" {
			if err := app.CreateFODTDocument(källorDir, docName, m.selectedFODTTemplate, assets.Files); err != nil {
				m.errMsg = fmt.Sprintf("Kunde inte skapa FODT-dokument: %v", err)
				return m, tea.Quit
			}
		}

		var mainCmd tea.Cmd
		m, mainCmd = m.goToMainMenu()
		cmds = append(cmds, mainCmd)
	}
	return m, tea.Batch(cmds...)
}

// ============================================
// DISPATCHER
// ============================================

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	}

	switch m.state {
	case stateMainMenu:
		return m.updateMainMenu(msg)
	case stateSettingsMenu:
		return m.updateSettingsMenu(msg)
	case stateSettingsCreateOrg:
		return m.updateSettingsCreateOrg(msg)
	case stateSettingsChooseOrgToEdit:
		return m.updateSettingsChooseOrgToEdit(msg)
	case stateSettingsEditOrgDetails:
		return m.updateSettingsEditOrgDetails(msg)
	case stateEditZip:
		return m.updateEditZip(msg)
	case stateEditOts:
		return m.updateEditOts(msg)
	case stateConfirmTemplateUpdate:
		return m.updateConfirmTemplateUpdate(msg)
	case stateChooseOrgForBuild:
		return m.updateChooseOrgForBuild(msg)
	case stateChooseDocumentToBuild:
		return m.updateChooseDocumentToBuild(msg)
	case stateAskIfUserWantToNukeSealedDocument:
		return m.updateAskNukeSealedDocument(msg)
	case stateChooseOrgForSeal:
		return m.updateChooseOrgForSeal(msg)
	case stateChooseDocumentToSeal:
		return m.updateChooseDocumentToSeal(msg)
	case stateAskIfUserWantToNukeSealedDocumentForSealedDocument:
		return m.updateAskNukeSealedForSeal(msg)
	case stateSelectKeyForSeal:
		return m.updateSelectKeyForSeal(msg)
	case stateEnterPinForSeal:
		return m.updateEnterPinForSeal(msg)
	case stateKeyManagementMenu:
		return m.updateKeyManagementMenu(msg)
	case stateGenerateKey:
		return m.updateGenerateKey(msg)
	case stateListKeys:
		return m.updateListKeys(msg)
	case stateChooseOrgForDoc:
		return m.updateChooseOrgForDoc(msg)
	case stateCreateOrganization:
		return m.updateCreateOrganization(msg)
	case stateFirstRun:
		return m.updateFirstRun(msg)
	case stateChooseDocumentTypeForm:
		return m.updateChooseDocumentType(msg)
	case stateChooseBodyForm:
		return m.updateChooseBody(msg)
	case stateCreateBody:
		return m.updateCreateBody(msg)
	case stateAskDateForm:
		return m.updateAskDate(msg)
	case stateChooseGoverningDocumentTypeForm:
		return m.updateChooseGoverningDocumentType(msg)
	case stateCreateNewGoverningDocumentTypeForm:
		return m.updateCreateNewGoverningDocumentType(msg)
	case stateAskGoverningDocumentNameForm:
		return m.updateAskGoverningDocumentName(msg)
	case stateChooseOtherDocumentType:
		return m.updateChooseOtherDocumentType(msg)
	case stateChooseOtherDocumentCategoryNameForm:
		return m.updateChooseOtherDocumentCategoryName(msg)
	case stateCreateOtherDocumentTypeForm:
		return m.updateCreateOtherDocumentType(msg)
	case stateExportPublicKey:
		return m.updateExportPublicKey(msg)
	case stateGenerateKeyContract:
		return m.updateGenerateKeyContract(msg)
	case stateKeyActionResult:
		return m.updateKeyActionResult(msg)
	case stateChooseSourceFormat:
		return m.updateChooseSourceFormat(msg)
	case stateChooseFODTTemplate:
		return m.updateChooseFODTTemplate(msg)
	default:
		m.errMsg = fmt.Sprintf("okänt tillstånd: %d", m.state)
		return m, tea.Quit
	}
}

// ============================================
// VIEW
// ============================================

func (m mainModel) View() string {
	switch m.state {
	case stateMainMenu:
		return m.mainMenu.View()
	case stateSettingsMenu:
		return m.settingsMenu.View()
	default:
		if m.activeForm == nil {
			return "Laddar..."
		}
		return m.activeForm.View()
	}
}

func StartTUI() error {
	cwd, _ := os.Getwd()
	cfg := app.Config{}

	initialModel := mainModel{
		state:        stateMainMenu,
		cwd:          cwd,
		cfg:          &cfg,
		mainMenu:     createMainMenuForm(),
		settingsMenu: createSettingsMenuForm(),
		activeForm:   createFirstRunForm(),
	}

	if isFirstRun(cwd) {
		initialModel.state = stateFirstRun
	} else {
		cfg, _ = app.LoadConfig(cwd)
	}

	p := tea.NewProgram(initialModel, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return err
	}
	if m, ok := finalModel.(mainModel); ok && m.errMsg != "" {
		return fmt.Errorf("%s", m.errMsg)
	}
	return nil
}
