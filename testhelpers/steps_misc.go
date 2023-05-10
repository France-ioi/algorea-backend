//go:build !prod

package testhelpers

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bou.ke/monkey"
	"github.com/cucumber/messages-go/v10"
	"github.com/go-chi/chi"
	"github.com/spf13/viper"
	"gorm.io/gorm"

	"github.com/France-ioi/AlgoreaBackend/app"
	"github.com/France-ioi/AlgoreaBackend/app/api/groups"
	"github.com/France-ioi/AlgoreaBackend/app/auth"
	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/token"
	"github.com/France-ioi/AlgoreaBackend/app/tokentest"
)

func (ctx *TestContext) IAmUserWithID(userID int64) error { //nolint
	ctx.userID = userID
	ctx.user = strconv.FormatInt(userID, 10)

	db, err := database.Open(ctx.db())
	if err != nil {
		return err
	}
	return database.NewDataStore(db).InTransaction(func(store *database.DataStore) error {
		store.Exec("SET FOREIGN_KEY_CHECKS=0")
		defer store.Exec("SET FOREIGN_KEY_CHECKS=1")
		return store.Sessions().InsertMap(map[string]interface{}{
			"access_token": testAccessToken,
			"user_id":      ctx.userID,
			"issued_at":    database.Now(),
			"expires_at":   gorm.Expr("? + INTERVAL 7200 SECOND", database.Now()),
		})
	})
}

func (ctx *TestContext) TimeNow(timeStr string) error { //nolint
	testTime, err := time.Parse(time.RFC3339Nano, timeStr)
	if err == nil {
		monkey.Patch(time.Now, func() time.Time { return testTime })
	}
	return err
}

func (ctx *TestContext) TimeIsFrozen() error { //nolint
	currentTime := time.Now()
	monkey.Patch(time.Now, func() time.Time { return currentTime })
	return nil
}

func (ctx *TestContext) TheGeneratedGroupCodeIs(generatedCode string) error { //nolint
	monkey.Patch(groups.GenerateGroupCode, func() (string, error) { return generatedCode, nil }) // nolint:unparam
	return nil
}

var multipleStringsRegexp = regexp.MustCompile(`^((?:\s*,\s*)?"([^"]*)")`)

func (ctx *TestContext) TheGeneratedGroupCodesAre(generatedCodes string) error { //nolint
	currentIndex := 0
	monkey.Patch(groups.GenerateGroupCode, func() (string, error) {
		currentIndex++
		code := multipleStringsRegexp.FindStringSubmatch(generatedCodes)
		if code == nil {
			return "", errors.New("not enough generated codes")
		}
		generatedCodes = generatedCodes[len(code[1]):]
		return code[2], nil
	})
	return nil
}

func (ctx *TestContext) TheGeneratedAuthKeyIs(generatedString string) error { //nolint
	monkey.Patch(auth.GenerateKey, func() (string, error) { return generatedString, nil }) // nolint:unparam
	return nil
}

func (ctx *TestContext) TheGeneratedAuthKeysAre(generatedStrings string) error { //nolint
	currentIndex := 0
	monkey.Patch(auth.GenerateKey, func() (string, error) {
		currentIndex++
		randomString := multipleStringsRegexp.FindStringSubmatch(generatedStrings)
		if randomString == nil {
			return "", errors.New("not enough generated random strings")
		}
		generatedStrings = generatedStrings[len(randomString[1]):]
		return randomString[2], nil
	})
	return nil
}

func (ctx *TestContext) LogsShouldContain(docString *messages.PickleStepArgument_PickleDocString) error { // nolint
	preprocessed, err := ctx.preprocessString(docString.Content)
	if err != nil {
		return err
	}
	stringToSearch := strings.TrimSpace(preprocessed)
	logs := ctx.logsHook.GetAllLogs()
	if !strings.Contains(logs, stringToSearch) {
		return fmt.Errorf("cannot find %q in logs:\n%s", stringToSearch, logs)
	}
	return nil
}

func (ctx *TestContext) SignedTokenIsDistributed(varName, signerName string, docString *messages.PickleStepArgument_PickleDocString) error { // nolint
	var privateKey *rsa.PrivateKey
	signerName = strings.TrimSpace(signerName)
	switch signerName {
	case "the app":
		config, _ := app.TokenConfig(ctx.application.Config)
		privateKey = config.PrivateKey
	case "the task platform":
		privateKey = tokentest.TaskPlatformPrivateKeyParsed
	default:
		return fmt.Errorf("unknown signer: %q. Only \"the app\" and \"the task platform\" are supported", signerName)
	}

	data, err := ctx.preprocessString(docString.Content)
	if err != nil {
		return err
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return err
	}
	ctx.templateSet.AddGlobal(varName, token.Generate(payload, privateKey))
	return nil
}

func (ctx *TestContext) TheApplicationConfigIs(body *messages.PickleStepArgument_PickleDocString) error { // nolint
	config := viper.New()
	config.SetConfigType("yaml")
	preprocessedConfig, err := ctx.preprocessString(body.Content)
	if err != nil {
		return err
	}
	err = config.MergeConfig(strings.NewReader(preprocessedConfig))
	if err != nil {
		return err
	}

	// Only 'domain' and 'auth' changes are currently supported
	if config.IsSet("auth") {
		ctx.application.ReplaceAuthConfig(config)
	}
	if config.IsSet("domains") {
		ctx.application.ReplaceDomainsConfig(config)
	}

	return nil
}

func (ctx *TestContext) TheContextVariableIs(variableName, value string) error { //nolint
	preprocessed, err := ctx.preprocessString(value)
	if err != nil {
		return err
	}

	oldHTTPHandler := ctx.application.HTTPHandler
	ctx.application.HTTPHandler = chi.NewRouter().With(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			//nolint:lll,golint,staticcheck SA1029: should not use built-in type string as key for value; define your own type to avoid collisions (staticcheck)
			oldHTTPHandler.ServeHTTP(writer, request.WithContext(context.WithValue(request.Context(), variableName, preprocessed)))
		})
	}).(*chi.Mux)
	ctx.application.HTTPHandler.Mount("/", oldHTTPHandler)
	return nil
}
