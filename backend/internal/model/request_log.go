package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// RequestLog records every request received by the mock engine.
type RequestLog struct {
	ID             uint              `gorm:"primaryKey" json:"_id,string"`
	ProjectID      uint              `gorm:"index;not null" json:"projectId,string"`
	APIID          uint              `gorm:"index" json:"apiId,string,omitempty"`
	Method         string            `gorm:"size:16" json:"method"`
	Path           string            `gorm:"size:512" json:"path"`
	HeadersJS      string            `gorm:"column:headers;type:text" json:"-"`
	BodyJS         string            `gorm:"column:body;type:text" json:"-"`
	QueryJS        string            `gorm:"column:query;type:text" json:"-"`
	ResponseStatus int               `json:"responseStatus"`
	ResponseBodyJS string            `gorm:"column:response_body;type:text" json:"-"`
	CreatedAt      time.Time         `json:"createdAt"`

	// Computed fields (not persisted).
	Headers      map[string]string `gorm:"-" json:"headers"`
	Body         any               `gorm:"-" json:"body"`
	Query        map[string]string `gorm:"-" json:"query"`
	ResponseBody any               `gorm:"-" json:"responseBody"`
}

// BeforeSave serializes dynamic fields.
func (l *RequestLog) BeforeSave(_ *gorm.DB) error {
	var err error
	if l.Headers != nil {
		b, e := json.Marshal(l.Headers)
		if e != nil {
			return e
		}
		l.HeadersJS = string(b)
	}
	if l.Body != nil {
		b, e := json.Marshal(l.Body)
		if e != nil {
			return e
		}
		l.BodyJS = string(b)
	}
	if l.Query != nil {
		b, e := json.Marshal(l.Query)
		if e != nil {
			return e
		}
		l.QueryJS = string(b)
	}
	if l.ResponseBody != nil {
		b, e := json.Marshal(l.ResponseBody)
		if e != nil {
			return e
		}
		l.ResponseBodyJS = string(b)
	}
	return err
}

// AfterFind restores dynamic fields.
func (l *RequestLog) AfterFind(_ *gorm.DB) error {
	l.Headers = map[string]string{}
	if l.HeadersJS != "" {
		_ = json.Unmarshal([]byte(l.HeadersJS), &l.Headers)
	}
	l.Query = map[string]string{}
	if l.QueryJS != "" {
		_ = json.Unmarshal([]byte(l.QueryJS), &l.Query)
	}
	if l.BodyJS != "" {
		var body any
		if json.Unmarshal([]byte(l.BodyJS), &body) == nil {
			l.Body = body
		} else {
			l.Body = l.BodyJS
		}
	}
	if l.ResponseBodyJS != "" {
		var body any
		if json.Unmarshal([]byte(l.ResponseBodyJS), &body) == nil {
			l.ResponseBody = body
		} else {
			l.ResponseBody = l.ResponseBodyJS
		}
	}
	return nil
}
