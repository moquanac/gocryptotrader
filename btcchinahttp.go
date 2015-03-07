package main

import (
	"net/http"
	"net/url"
	"strconv"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"io/ioutil"
	"fmt"
	"log"
)

const (
	BTCCHINA_API_URL = "https://api.btcchina.com/"
)

type BTCChina struct {
	Name string
	Enabled bool
	Verbose bool
	APISecret, APIKey string
	Fee float64
}

type BTCChinaTicker struct {
	High float64 `json:",string"`
	Low float64 `json:",string"`
	Buy float64 `json:",string"`
	Sell float64 `json:",string"`
	Last float64 `json:",string"`
	Vol float64 `json:",string"`
	Date int64
	Vwap float64 `json:",string"`
	Prev_close float64 `json:",string"`
	Open float64 `json:",string"`
}

func (b *BTCChina) SetDefaults() {
	b.Name = "BTC China"
	b.Enabled = true
	b.Fee = 0
	b.Verbose = false
}

func (b *BTCChina) GetName() (string) {
	return b.Name
}

func (b *BTCChina) SetEnabled(enabled bool) {
	b.Enabled = enabled
}

func (b *BTCChina) IsEnabled() (bool) {
	return b.Enabled
}

func (b *BTCChina) SetAPIKeys(apiKey, apiSecret string) {
	b.APIKey = apiKey
	b.APISecret = apiSecret
}

func (b *BTCChina) GetFee() (float64) {
	return b.Fee
}

func (b *BTCChina) GetTicker(symbol string) (BTCChinaTicker) {
	type Response struct {
		Ticker BTCChinaTicker
	}

	resp := Response{}
	req := fmt.Sprintf("%sdata/ticker?market=%s", BTCCHINA_API_URL, symbol)
	err := SendHTTPRequest(req, true, &resp)
	if err != nil {
		log.Println(err)
		return BTCChinaTicker{}
	}
	return resp.Ticker
}

func (b *BTCChina) GetTradesLast24h(symbol string) (bool) {
	req := fmt.Sprintf("%sdata/trades?market=%s", BTCCHINA_API_URL, symbol)
	err := SendHTTPRequest(req, true, nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (b *BTCChina) GetTradeHistory(symbol string, limit, sinceTid int64, time time.Time) (bool) {
	req := fmt.Sprintf("%sdata/historydata?market=%s", BTCCHINA_API_URL, symbol)
	v := url.Values{}

	if limit > 0 {
		v.Set("limit", strconv.FormatInt(limit, 10))
	}
	if sinceTid > 0 {
		v.Set("since", strconv.FormatInt(sinceTid, 10))
	}
	if !time.IsZero() {
		v.Set("sincetype", strconv.FormatInt(time.Unix(), 10))
	}

	values := v.Encode()
	if (len(values) > 0) {
		req += "?" + values
	}

	err := SendHTTPRequest(req, true, nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (b *BTCChina) GetOrderBook(symbol string, limit int) (bool) {
	req := fmt.Sprintf("%sdata/orderbook?market=%s&limit=%d", BTCCHINA_API_URL, symbol, limit)
	err := SendHTTPRequest(req, true, nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (b *BTCChina) GetAccountInfo() {
	err := b.SendAuthenticatedHTTPRequest("getAccountInfo", nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCChina) BuyOrder(price, amount float64) {
	params := []string{}

	if (price != 0) {
		params = append(params, strconv.FormatFloat(price, 'f', 2, 64))
	}

	if (amount != 0) {
		params = append(params, strconv.FormatFloat(amount, 'f', 8, 64))
	}

	err := b.SendAuthenticatedHTTPRequest("buyOrder2", params)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCChina) SendAuthenticatedHTTPRequest(method string, params []string) (err error) {
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)[0:16]
	paramsEncoded := ""

	if (len(params) == 0) {
		params = []string{}
	} else {
		paramsEncoded = strings.Join(params, ",")
	}

	encoded := fmt.Sprintf("tonce=%s&accesskey=%s&requestmethod=post&id=%d&method=%s&params=%s", nonce, b.APIKey, 1, method, paramsEncoded)

	if b.Verbose {
		log.Println(encoded)
	}

	hmac := GetHMAC(sha1.New, []byte(encoded), []byte(b.APISecret))
	postData := make(map[string]interface{})
	postData["method"] = method
	postData["params"] = params
	postData["id"] = 1
	data, err := json.Marshal(postData)

	if err != nil {
		return errors.New("Unable to JSON POST data")
	}

	if b.Verbose {
		log.Printf("Sending POST request to %s calling method %s with params %s\n", "https://api.btcchina.com/api_trade_v1.php", method, data)
	}

	req, err := http.NewRequest("POST", "https://api.btcchina.com/api_trade_v1.php", strings.NewReader(string(data)))

	if err != nil {
		return err
	}

	req.Header.Add("Content-type", "application/json-rpc")
	req.Header.Add("Authorization", "Basic " + Base64Encode([]byte(b.APIKey + ":" + HexEncodeToString(hmac))))
	req.Header.Add("Json-Rpc-Tonce", nonce)

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return errors.New("PostRequest: Unable to send request")
	}

	contents, _ := ioutil.ReadAll(resp.Body)
	
	if b.Verbose {
		log.Printf("Recv'd :%s\n", string(contents))
	}

	resp.Body.Close()
	return nil

}