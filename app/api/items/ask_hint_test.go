package items

import (
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"

	"github.com/France-ioi/AlgoreaBackend/app/database"
	"github.com/France-ioi/AlgoreaBackend/app/formdata"
	"github.com/France-ioi/AlgoreaBackend/app/payloads"
	"github.com/France-ioi/AlgoreaBackend/app/payloadstest"
	"github.com/France-ioi/AlgoreaBackend/app/token"
	"github.com/France-ioi/AlgoreaBackend/app/tokentest"
)

func TestAskHintRequest_UnmarshalJSON(t *testing.T) {
	expectedTaskToken := token.Task{}
	_ = payloads.ParseMap(payloadstest.TaskPayloadFromAlgoreaPlatform, &expectedTaskToken)
	expectedTaskToken.Converted.UserID = 556371821693219925
	expectedTaskToken.Converted.LocalItemID = 901756573345831409
	expectedTaskToken.Converted.AttemptID = 100
	expectedTaskToken.Converted.ParticipantID = 556371821693219925
	expectedHintToken := token.Hint{}
	expectedHintToken.Converted.UserID = 556371821693219925
	_ = payloads.ParseMap(payloadstest.HintPayloadFromTaskPlatform, &expectedHintToken)
	monkey.Patch(time.Now,
		func() time.Time { return time.Date(2019, 5, 2, 12, 0, 0, 0, time.UTC) })
	defer monkey.UnpatchAll()

	type platform struct {
		publicKey string
	}
	tests := []struct {
		name     string
		platform *platform
		raw      []byte
		expected AskHintRequest
		wantErr  error
		mockDB   bool
		itemID   int64
	}{
		{
			name:    "invalid json",
			raw:     []byte(""),
			wantErr: errors.New("unexpected end of JSON input"),
		},
		{
			name:    "missing task_token",
			raw:     []byte(`{}`),
			wantErr: errors.New("missing task_token"),
		},
		{
			name:    "invalid task_token",
			raw:     []byte(`{"task_token":"ABC.DEF.ABC"}`),
			wantErr: errors.New("invalid task_token: invalid character '\\x00' looking for beginning of value"),
		},
		{
			name: "missing hint_requested token",
			raw: []byte(fmt.Sprintf(`{"task_token": %q}`,
				token.Generate(payloadstest.TaskPayloadFromAlgoreaPlatform, tokentest.AlgoreaPlatformPrivateKeyParsed))),
			wantErr: errors.New("missing hint_requested"),
		},
		{
			name: "missing platform",
			raw: []byte(fmt.Sprintf(`{"task_token": %q, "hint_requested": "ABC.DEF.ABC"}`,
				token.Generate(payloadstest.TaskPayloadFromAlgoreaPlatform, tokentest.AlgoreaPlatformPrivateKeyParsed))),
			wantErr: errors.New("cannot find the platform for item 901756573345831409"),
			mockDB:  true,
			itemID:  901756573345831409,
		},
		{
			name: "wrong platform's public key",
			raw: []byte(fmt.Sprintf(`{"task_token": %q, "hint_requested": "ABC.DEF.ABC"}`,
				token.Generate(payloadstest.TaskPayloadFromAlgoreaPlatform, tokentest.AlgoreaPlatformPrivateKeyParsed))),
			wantErr:  errors.New("invalid hint_requested: wrong platform's key"),
			mockDB:   true,
			itemID:   901756573345831409,
			platform: &platform{publicKey: "zzz"},
		},
		{
			name: "hint_requested is not a string, but it should be a token",
			raw: []byte(fmt.Sprintf(`{"task_token": %q, "hint_requested": 1234}`,
				token.Generate(payloadstest.TaskPayloadFromAlgoreaPlatform, tokentest.AlgoreaPlatformPrivateKeyParsed))),
			wantErr:  errors.New("invalid hint_requested: json: cannot unmarshal number into Go value of type string"),
			mockDB:   true,
			itemID:   901756573345831409,
			platform: &platform{publicKey: string(tokentest.AlgoreaPlatformPublicKey)},
		},
		{
			name: "invalid hint_requested token",
			raw: []byte(fmt.Sprintf(`{"task_token": %q, "hint_requested": "ABC.DEF.ABC"}`,
				token.Generate(payloadstest.TaskPayloadFromAlgoreaPlatform, tokentest.AlgoreaPlatformPrivateKeyParsed))),
			wantErr:  errors.New("invalid hint_requested: invalid character '\\x00' looking for beginning of value"),
			mockDB:   true,
			platform: &platform{publicKey: string(tokentest.AlgoreaPlatformPublicKey)},
			itemID:   901756573345831409,
		},
		{
			name: "everything is okay",
			raw: []byte(fmt.Sprintf(`{"task_token": %q, "hint_requested": %q}`,
				token.Generate(payloadstest.TaskPayloadFromAlgoreaPlatform, tokentest.AlgoreaPlatformPrivateKeyParsed),
				token.Generate(payloadstest.HintPayloadFromTaskPlatform, tokentest.TaskPlatformPrivateKeyParsed),
			)),
			mockDB:   true,
			platform: &platform{publicKey: string(tokentest.TaskPlatformPublicKey)},
			itemID:   901756573345831409,
			expected: AskHintRequest{
				TaskToken: &expectedTaskToken,
				HintToken: &expectedHintToken,
			},
		},
		{
			name: "plain hint_requested is not a map",
			raw: []byte(fmt.Sprintf(`{"task_token": %q, "hint_requested": []}`,
				token.Generate(payloadstest.TaskPayloadFromAlgoreaPlatform, tokentest.AlgoreaPlatformPrivateKeyParsed),
			)),
			mockDB:   true,
			itemID:   901756573345831409,
			platform: &platform{},
			wantErr: errors.New("invalid hint_requested: " +
				"json: cannot unmarshal array into Go value of type map[string]formdata.Anything"),
		},
		{
			name: "invalid plain hint_requested",
			raw: []byte(fmt.Sprintf(`{"task_token": %q, "hint_requested": {"someField":"value"}}`,
				token.Generate(payloadstest.TaskPayloadFromAlgoreaPlatform, tokentest.AlgoreaPlatformPrivateKeyParsed),
			)),
			mockDB:   true,
			itemID:   901756573345831409,
			platform: &platform{},
			wantErr:  errors.New("invalid hint_requested: invalid HintToken: invalid input data"),
		},
		{
			name: "plain hint_requested is okay",
			raw: []byte(fmt.Sprintf(`{"task_token": %q, "hint_requested": {"idUser":"556371821693219925","askedHint":"123"}}`,
				token.Generate(payloadstest.TaskPayloadFromAlgoreaPlatform, tokentest.AlgoreaPlatformPrivateKeyParsed),
			)),
			mockDB:   true,
			itemID:   901756573345831409,
			platform: &platform{},
			expected: AskHintRequest{
				TaskToken: &expectedTaskToken,
				HintToken: &token.Hint{
					UserID:    "556371821693219925",
					AskedHint: *formdata.AnythingFromString(`"123"`),
					Converted: payloads.HintTokenConverted{
						UserID: 556371821693219925,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			db, mock := database.NewDBMock()
			defer func() { _ = db.Close() }()

			if tt.mockDB {
				mockQuery := mock.ExpectQuery(regexp.QuoteMeta("SELECT `public_key` " +
					"FROM `platforms` JOIN items ON items.platform_id = platforms.id WHERE items.id = ?")).
					WithArgs(tt.itemID)

				if tt.platform != nil {
					publicKey := &tt.platform.publicKey
					if tt.platform.publicKey == "" {
						publicKey = nil
					}
					mockQuery.
						WillReturnRows(mock.NewRows([]string{"public_key"}).AddRow(publicKey))
				} else {
					mockQuery.
						WillReturnRows(mock.NewRows([]string{"public_key"}))
				}
			}
			r := &AskHintRequest{
				store:     database.NewDataStore(db),
				publicKey: tokentest.AlgoreaPlatformPublicKeyParsed,
			}
			err := r.UnmarshalJSON(tt.raw)
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				if err == nil {
					assert.Equal(t, tt.wantErr, err)
				} else {
					assert.Equal(t, tt.wantErr.Error(), err.Error())
				}
			}
			if err == nil {
				r.store = nil
				if r.HintToken != nil {
					r.HintToken.PublicKey = nil
				}
				if r.TaskToken != nil {
					r.TaskToken.PublicKey = nil
				}
				r.publicKey = nil
				assert.Equal(t, &tt.expected, r)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
