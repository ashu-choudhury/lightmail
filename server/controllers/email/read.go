package email

import (
	"encoding/json"
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/dto/response"
	"github.com/Jinnrry/pmail/models"
	"github.com/Jinnrry/pmail/utils/context"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
)

type markReadRequest struct {
	IDs    []int `json:"ids"`
	IsRead *bool `json:"isRead"`
}

func MarkRead(ctx *context.Context, w http.ResponseWriter, req *http.Request) {
	reqBytes, err := io.ReadAll(req.Body)
	if err != nil {
		log.WithContext(ctx).Errorf("%+v", err)
	}
	var reqData markReadRequest
	err = json.Unmarshal(reqBytes, &reqData)
	if err != nil {
		log.WithContext(ctx).Errorf("%+v", err)
	}

	if len(reqData.IDs) <= 0 {
		response.NewErrorResponse(response.ParamsError, "IDs required", "").FPrint(w)
		return
	}

	targetReadStatus := int8(1)
	if reqData.IsRead != nil && !*reqData.IsRead {
		targetReadStatus = int8(0)
	}

	for _, id := range reqData.IDs {
		var ue models.UserEmail
		has, err := db.Instance.Where("email_id = ? AND user_id = ?", id, ctx.UserID).Get(&ue)
		if err != nil {
			log.WithContext(ctx).Errorf("SQL error: %+v", err)
			continue
		}
		if has {
			ue.IsRead = targetReadStatus
			_, err = db.Instance.Where("id = ?", ue.ID).Cols("is_read").Update(&ue)
			if err != nil {
				log.WithContext(ctx).Errorf("SQL error updating is_read: %+v", err)
			}
		}
	}

	response.NewSuccessResponse("success").FPrint(w)
}
