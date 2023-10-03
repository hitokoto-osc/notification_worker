package model

import (
	"context"
	lib "github.com/go-playground/validator/v10"
	"github.com/hitokoto-osc/notification-worker/consumers/notification/v1/internal/vcarbon"
	"github.com/hitokoto-osc/notification-worker/utils/validator"
	"github.com/stretchr/testify/assert"
	"testing"
)

// 本测试文件用于测试消息模型的各个校验器是否正常工作。

func TestHitokotoAppendedMessageNormal(t *testing.T) {
	var buff = []byte(`{"to": "a632079@qq.com","uuid": "701fd013-65eb-4813-85e3-0f74bc53a95f", "hitokoto": "test", "type": "a", "from": "b", "from_who": null, "creator": "d", "created_at": "1696347595"}`)
	data, err := validator.UnmarshalV[HitokotoAppendedMessage](context.Background(), buff)

	assert.NoError(t, err)
	t.Logf("%+v", data)
}

func TestTestHitokotoAppendedMessageUUIDIsIncorrect(t *testing.T) {
	var buff = []byte(`{"to": "a632079@qq.com","uuid": "7-65eb-4813-85e3-0f74bc53a95f", "hitokoto": "test", "type": "a", "from": "b", "from_who": null, "creator": "d", "created_at": "1696347595"}`)
	_, err := validator.UnmarshalV[HitokotoAppendedMessage](context.Background(), buff)
	var e lib.ValidationErrors
	assert.ErrorAs(t, err, &e)
}

func TestTestHitokotoAppendedMessageTypeIsIncorrect(t *testing.T) {
	var buff = []byte(`{"to": "a632079@qq.com","uuid": "701fd013-65eb-4813-85e3-0f74bc53a95f", "hitokoto": "test", "type": "zXca", "from": "b", "from_who": null, "creator": "d", "created_at": "1696347595"}`)
	_, err := validator.UnmarshalV[HitokotoAppendedMessage](context.Background(), buff)
	var e lib.ValidationErrors
	assert.ErrorAs(t, err, &e)
}

func TestTestHitokotoAppendedMessageCreatedAtIsInCorrect(t *testing.T) {
	var buff = []byte(`{"to": "a632079@qq.com","uuid": "701fd013-65eb-4813-85e3-0f74bc53a95f", "hitokoto": "test", "type": "a", "from": "b", "from_who": null, "creator": "d", "created_at": "WWWWW"}`)
	_, err := validator.UnmarshalV[HitokotoAppendedMessage](context.Background(), buff)
	var e *vcarbon.InvalidCarbonParseParameterError
	assert.ErrorAs(t, err, &e)
}

// TODO: 补完其他内容的测试
