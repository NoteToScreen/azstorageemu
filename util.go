package main

import (
	"encoding/xml"
	"time"
)

// TimeRFC1123 is an alias for time.Time needed for custom Unmarshalling
type TimeRFC1123 time.Time

// UnmarshalXML is a custom unmarshaller that overrides the default time unmarshal which uses a different time layout.
func (t *TimeRFC1123) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var value string
	d.DecodeElement(&value, &start)
	parse, err := time.Parse(time.RFC1123, value)
	if err != nil {
		return err
	}
	*t = TimeRFC1123(parse)
	return nil
}

func (t *TimeRFC1123) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	e.EncodeElement(time.Time(*t).Format(time.RFC1123), start)
	return nil
}
