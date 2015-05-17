package conquest

import (
	"conquest/utils"
	"encoding/json"
	"errors"
	"github.com/robertkrimen/otto"
	"net/url"
	"os"
	"time"
)

// Returns t as otto.Value or panics
func toOttoValueOrPanic(vm *otto.Otto, t interface{}) otto.Value {
	obj, err := vm.ToValue(t)
	utils.UnlessNilThenPanic(err)

	return obj
}

// javascript conquest object
type JSConquest struct {
	conquest *Conquest
	vm       *otto.Otto
}

// conquest.prototype.Dump
// Prints configuration by javascript and panics
// Useful for debugging
// Ex:
// conquest.Dump()
func (c JSConquest) Dump(call otto.FunctionCall) otto.Value {
	jbyte, err := json.MarshalIndent(c.conquest, "", "\t")
	utils.UnlessNilThenPanic(err)

	panic(string(jbyte))
	return otto.Value{}
}

// conquest.prototype.Proto
// Sets HTTP protocol
// Ex: conquest.Proto("HTTP/1.1");
func (c JSConquest) Proto(call otto.FunctionCall) otto.Value {
	proto, err := call.Argument(0).ToString()
	utils.UnlessNilThenPanic(err)

	if proto != "HTTP/1.1" && proto != "HTTP/1.0" {
		panic("Only HTTP/1.1 and HTTP/1.0 protocols are available.")
	}
	c.conquest.Proto = proto
	return toOttoValueOrPanic(c.vm, c)
}

// conquest.prototype.Insecure(insecure)
// Sets Conquest struct's TlsInsecure boolean value.
// Use it for skip verify at secure https connections
// Ex:
// conquest.Insecure(true);
func (c JSConquest) Insecure(call otto.FunctionCall) otto.Value {
	insecure, err := call.Argument(0).ToBoolean()
	utils.UnlessNilThenPanic(err)

	c.conquest.TlsInsecure = insecure
	return toOttoValueOrPanic(c.vm, c)
}

// conquest.prototype.Host(host)
// Sets Host info ( and header )
// Ex: conquest.Host("mydomain.local:3434");
func (c JSConquest) Host(call otto.FunctionCall) otto.Value {
	host_str, err := call.Argument(0).ToString()
	utils.UnlessNilThenPanic(err)

	hostUrl, err := url.Parse(host_str)
	utils.UnlessNilThenPanic(err)

	c.conquest.Host = hostUrl.Host
	return toOttoValueOrPanic(c.vm, c)
}

// conquest.prototype.PemFile
// Sets pem file for ssl connections
// Ex:
// conquest.PemFile("/path/to/file.pem")
func (c JSConquest) PemFile(call otto.FunctionCall) otto.Value {
	pemfile, err := call.Argument(0).ToString()
	utils.UnlessNilThenPanic(err)

	_, err = os.Stat(pemfile)
	utils.UnlessNilThenPanic(err)

	c.conquest.PemFilePath = pemfile

	return toOttoValueOrPanic(c.vm, c)
}

// conquest.prototype.Duration
// Sets the duration of tests
// Ex:
// conquest.Duration("10m")
func (c JSConquest) Duration(call otto.FunctionCall) otto.Value {
	durationStr, err := call.Argument(0).ToString()
	utils.UnlessNilThenPanic(err)

	duration, err := time.ParseDuration(durationStr)
	utils.UnlessNilThenPanic(err)

	c.conquest.Duration = &duration

	return toOttoValueOrPanic(c.vm, c)
}

// conquest.prototype.Requests(reqs_count)
// Sets the total requests count
// Ex: conquest.Requests(1000);
func (c JSConquest) Requests(call otto.FunctionCall) otto.Value {
	reqs, err := call.Argument(0).ToInteger()
	utils.UnlessNilThenPanic(err)

	c.conquest.TotalRequests = reqs

	return toOttoValueOrPanic(c.vm, c)
}

// sets initial cookies and headers for conquest
func conquestInitials(conquest *Conquest, method string, call *otto.FunctionCall) {
	arg := call.Argument(0)
	panicStr := method + " function parameter 1 must be an object."

	if arg.Class() != "Object" {
		panic(errors.New(panicStr))
	}

	argObj := arg.Object()
	if argObj == nil {
		panic(errors.New(panicStr))
	}

	for _, k := range argObj.Keys() {
		val, err := argObj.Get(k)
		if err != nil {
			continue
		}

		valStr, err := val.ToString()
		if err != nil {
			continue
		}

		if _, exists := conquest.Initials[method]; !exists {
			conquest.Initials[method] = map[string]string{}
		}

		conquest.Initials[method][k] = valStr
	}
}

// conquest.prototype.Headers
// Sets initial headers which will be used for every request
// Ex:
// conquest.Headers({"X-Header1": "Conquest", "X-Header2": "Nothing"})
func (c JSConquest) Headers(call otto.FunctionCall) otto.Value {
	conquestInitials(c.conquest, "Headers", &call)
	return toOttoValueOrPanic(c.vm, c)
}

// conquest.prototype.Cookies
// Sets initial cookies which will be used for every request
// Ex:
// conquest.Cookies({"name":"value"})
func (c JSConquest) Cookies(call otto.FunctionCall) otto.Value {
	conquestInitials(c.conquest, "Cookies", &call)
	return toOttoValueOrPanic(c.vm, c)
}

// conquest.prototype.Cookies
// Sets total user count and calls user defined functions with JSTransactionCtx
// Ex: conquest.Users(100, function(user){})
func (c JSConquest) Users(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) != 2 {
		panic(errors.New("conquest.Users method takes exactly 2 arguments."))
	}

	uc, err := call.Argument(0).ToInteger()
	utils.UnlessNilThenPanic(err)

	c.conquest.TotalUsers = uc

	fn := call.Argument(1)
	if !fn.IsFunction() {
		panic(errors.New("Users function argument 2 must be a function."))
	}

	ctx := NewJSTransactionCtx(&c)
	ctxObj := toOttoValueOrPanic(c.vm, *ctx)

	_, err = fn.Call(fn, ctxObj)
	utils.UnlessNilThenPanic(err)

	return toOttoValueOrPanic(c.vm, c)
}

// Transaction context manager
type JSTransactionCtx struct {
	jsconquest *JSConquest
}

// Returns new JSTransactionCtx pointer
func NewJSTransactionCtx(jsc *JSConquest) *JSTransactionCtx {
	return &JSTransactionCtx{
		jsconquest: jsc,
	}
}

// Adds or gets context from Conquest.Track
// If wanted ctx type is Finally, function goes to last of Track and
// uses it if its type is Finally, otherwise adds a new Finally type ctx
// if wanted ctx type is Every/Cases/Then, function goes to last of Track
// and adds a new ctx if the last ctx is not Finally, otherwise breaks the link
// of it then links it to new ctx.
// And finally, calls user-defined function with JSTransaction parameters in the
// new ctx.
func ctxResolve(ctxType uint8, jsctx *JSTransactionCtx,
	call *otto.FunctionCall) otto.Value {

	fn := call.Argument(0)
	if !fn.IsFunction() {
		panic(errors.New("Context functions argument 1 must be a function."))
	}

	track := jsctx.jsconquest.conquest.Track
	var ctx *TransactionContext
	if track == nil || track.CtxType == CTX_FINALLY {
		ctx = &TransactionContext{
			CtxType: ctxType,
		}
		if track != nil {
			ctx.Next = track
		}
		jsctx.jsconquest.conquest.Track = ctx
		goto CALL_UD_FN
	}

	for ; track.Next != nil &&
		track.Next.CtxType != CTX_FINALLY; track = track.Next {
	}

	switch ctxType {
	case CTX_FINALLY:
		if track.Next != nil {
			ctx = track.Next
			break
		}

		ctx = &TransactionContext{
			CtxType: CTX_FINALLY,
		}
		track.Next = ctx

	case CTX_EVERY:
		fallthrough
	case CTX_THEN:
		ctx = &TransactionContext{
			CtxType: ctxType,
		}
		if track.Next != nil {
			ctx.Next = track.Next
		}
		track.Next = ctx
	}

CALL_UD_FN:
	jstact := &JSTransaction{
		jsconquest: jsctx.jsconquest,
		ctx:        ctx,
		Response: JSTransactionResponse{
			jsconquest: jsctx.jsconquest,
		},
	}
	jstact_obj := toOttoValueOrPanic(jsctx.jsconquest.vm, *jstact)
	_, err := fn.Call(fn, jstact_obj)
	utils.UnlessNilThenPanic(err)

	return toOttoValueOrPanic(jsctx.jsconquest.vm, *jsctx)
}

// users.Every
func (c JSTransactionCtx) Every(call otto.FunctionCall) otto.Value {
	return ctxResolve(CTX_EVERY, &c, &call)
}

// users.Then
func (c JSTransactionCtx) Then(call otto.FunctionCall) otto.Value {
	return ctxResolve(CTX_THEN, &c, &call)
}

// users.Cases
func (c JSTransactionCtx) Cases(call otto.FunctionCall) otto.Value {
	return ctxResolve(CTX_THEN, &c, &call)
}

// users.Finally
func (c JSTransactionCtx) Finally(call otto.FunctionCall) otto.Value {
	return ctxResolve(CTX_FINALLY, &c, &call)
}

type JSTransactionResponse struct {
	jsconquest  *JSConquest
	transaction *Transaction
}

// Inserts a map as like "StatusCode":[code] into transactions response
// conditions
// Ex: t.Response.StatusCode(200)
func (r JSTransactionResponse) StatusCode(call otto.FunctionCall) otto.Value {
	code, err := call.Argument(0).ToInteger()
	utils.UnlessNilThenPanic(err)

	r.transaction.ResConditions["StatusCode"] = code
	return toOttoValueOrPanic(r.jsconquest.vm, r)
}

// Inserts a map as like "Contains":[substr] into transactions response
// conditions
// Ex: t.Response.Contains("<h1>Fancy Header</h1>")
func (r JSTransactionResponse) Contains(call otto.FunctionCall) otto.Value {
	substr, err := call.Argument(0).ToString()
	utils.UnlessNilThenPanic(err)

	r.transaction.ResConditions["Contains"] = substr
	return toOttoValueOrPanic(r.jsconquest.vm, r)
}

// Inserts a map as like [name]:[expected] into kind map of
// transactions response conditions. if conditions[kind] is not allocated,
// allocates first.
func expectedAdditionals(kind string, call *otto.FunctionCall,
	r *JSTransactionResponse) otto.Value {
	if len(call.ArgumentList) != 2 {
		panic(errors.New("Response." + kind +
			" function takes exactly 2 arguments."))
	}

	name, err := call.Argument(0).ToString()
	utils.UnlessNilThenPanic(err)

	expected, err := call.Argument(1).ToString()
	utils.UnlessNilThenPanic(err)

	if _, exists := r.transaction.ResConditions[kind]; !exists {
		r.transaction.ResConditions[kind] = map[string]string{}
	}
	r.transaction.ResConditions[kind].(map[string]string)[name] = expected
	return toOttoValueOrPanic(r.jsconquest.vm, *r)
}

// Sets expected headers
// Ex: t.Response.Header("X-Header", "Expected Value");
func (r JSTransactionResponse) Header(call otto.FunctionCall) otto.Value {
	return expectedAdditionals("Header", &call, &r)
}

// Sets expected cookies
// Ex: t.Response.Cookie("X-Header", "Expected Value");
func (r JSTransactionResponse) Cookie(call otto.FunctionCall) otto.Value {
	return expectedAdditionals("Cookie", &call, &r)
}

// Transaction methods which called by passed as an argument at the context
// functions.
type JSTransaction struct {
	jsconquest  *JSConquest
	ctx         *TransactionContext
	transaction *Transaction
	Response    JSTransactionResponse
}

func (t *JSTransaction) unlessAllocatedThenPanic() {
	if t.transaction == nil {
		panic(errors.New("Call Do function first for allocate a new http request."))
	}
}

// Creates new transaction
// Ex: var t = user.Do("GET", "/")
func (t JSTransaction) Do(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) != 2 {
		panic(errors.New("Do function takes exactly 2 parameters."))
	}

	verb, err := call.Argument(0).ToString()
	utils.UnlessNilThenPanic(err)

	path, err := call.Argument(1).ToString()
	utils.UnlessNilThenPanic(err)

	t.transaction = &Transaction{
		conquest:      t.jsconquest.conquest,
		Verb:          verb,
		Path:          path,
		Headers:       map[string]string{},
		Cookies:       map[string]string{},
		ResConditions: map[string]interface{}{},
		Body:          map[string]interface{}{},
	}

	if t.ctx.Transactions == nil {
		t.ctx.Transactions = []*Transaction{}
	}

	t.Response.transaction = t.transaction
	t.ctx.Transactions = append(t.ctx.Transactions, t.transaction)
	return toOttoValueOrPanic(t.jsconquest.vm, t)
}

// Sets ReqOptions as clear initial cookies and headers
// Ex: t.ClearInitials()
func (t JSTransaction) ClearInitials(call otto.FunctionCall) otto.Value {
	t.unlessAllocatedThenPanic()
	t.transaction.ReqOptions |= CLEAR_COOKIES | CLEAR_HEADERS
	return toOttoValueOrPanic(t.jsconquest.vm, t)
}

// Sets ReqOptions as clear initial headers
// Ex: t.ClearHeaders()
func (t JSTransaction) ClearHeaders(call otto.FunctionCall) otto.Value {
	t.unlessAllocatedThenPanic()
	t.transaction.ReqOptions |= CLEAR_HEADERS
	return toOttoValueOrPanic(t.jsconquest.vm, t)
}

// Sets ReqOptions as clear initial cookies
// Ex: t.ClearCookies()
func (t JSTransaction) ClearCookies(call otto.FunctionCall) otto.Value {
	t.unlessAllocatedThenPanic()
	t.transaction.ReqOptions |= CLEAR_COOKIES
	return toOttoValueOrPanic(t.jsconquest.vm, t)
}

// Sets ReqOptions as reject cookies after http request
// Ex: t.RejectCookies()
func (t JSTransaction) RejectCookies(call otto.FunctionCall) otto.Value {
	t.unlessAllocatedThenPanic()
	t.transaction.ReqOptions |= REJECT_COOKIES
	return toOttoValueOrPanic(t.jsconquest.vm, t)
}

// Sets additional cookies and headers
func setAdditionals(kind string, call *otto.FunctionCall,
	t *JSTransaction) otto.Value {
	t.unlessAllocatedThenPanic()
	if len(call.ArgumentList) != 2 {
		panic(errors.New("Set" + kind + " function takes exactly 2 arguments."))
	}
	key, err := call.Argument(0).ToString()
	utils.UnlessNilThenPanic(err)

	val, err := call.Argument(1).ToString()
	utils.UnlessNilThenPanic(err)

	var hMap map[string]string
	switch kind {
	case "Header":
		hMap = t.transaction.Headers
	case "Cookie":
		hMap = t.transaction.Cookies
	}

	hMap[key] = val

	return toOttoValueOrPanic(t.jsconquest.vm, *t)
}

// Sets additional headers
// Ex: t.SetHeader("X-HeaderName", "HeaderValue")
func (t JSTransaction) SetHeader(call otto.FunctionCall) otto.Value {
	return setAdditionals("Header", &call, &t)
}

// Sets additional cookies
// Ex: t.SetCookie("name", "value")
func (t JSTransaction) SetCookie(call otto.FunctionCall) otto.Value {
	return setAdditionals("Cookie", &call, &t)
}

// Sets request body or query
// Ex: t.Body({
// "field1": "value",
// "field2":"value",
// "field3": function(fetch){ return fetch.FromDisk("/path", "mime-type"); },
// })
func (t JSTransaction) Body(call otto.FunctionCall) otto.Value {
	t.unlessAllocatedThenPanic()

	arg := call.Argument(0)
	panicStr := "Body function parameter 1 must be an object."

	if arg.Class() != "Object" {
		panic(errors.New(panicStr))
	}

	argObj := arg.Object()
	if argObj == nil {
		panic(errors.New(panicStr))
	}

	for _, k := range argObj.Keys() {
		val, err := argObj.Get(k)
		if err != nil {
			continue
		}

		if val.IsFunction() {
			fetch := &JSFetch{
				jsconquest: t.jsconquest,
			}
			jsf := toOttoValueOrPanic(t.jsconquest.vm, *fetch)
			retfn, err := val.Call(val, jsf)
			if err != nil {
				continue
			}
			exp, err := retfn.Export()
			if err != nil {
				continue
			}

			notation, err := mapToFetchNotation(exp.(map[string]interface{}))
			if err != nil {
				continue
			}
			t.transaction.Body[k] = notation

			continue
		}

		valStr, err := val.ToString()
		if err == nil {
			t.transaction.Body[k] = valStr
			continue
		}
	}

	return toOttoValueOrPanic(t.jsconquest.vm, t)
}

// fetch object which will be passed as argument user-defined argument at 
// body, header, cookies functions.
type JSFetch struct {
	jsconquest *JSConquest
}

// fetch.Fetch* methods
// returns FetchNotation
func fetchFrom(kind uint8, call *otto.FunctionCall,
	f *JSFetch) otto.Value {
	args := []string{}
	for _, v := range call.ArgumentList {
		vstr, err := v.ToString()
		if err != nil {
			continue
		}
		args = append(args, vstr)
	}

	notation := map[string]interface{}{
		"type": kind,
		"args": args,
	}

	return toOttoValueOrPanic(f.jsconquest.vm, notation)
}

// fetch.FromHeader
// Ex: fetch.FromHeader("X-HeaderName")
func (f JSFetch) FromHeader(call otto.FunctionCall) otto.Value {
	return fetchFrom(FETCH_HEADER, &call, &f)
}

// fetch.FromCookie
// ex: fetch.FromCookie("cookie_name")
func (f JSFetch) FromCookie(call otto.FunctionCall) otto.Value {
	return fetchFrom(FETCH_COOKIE, &call, &f)
}

// fetch.FromDisk
// ex: fetch.FromDisk("/path/to/files/", "mime-type")
func (f JSFetch) FromDisk(call otto.FunctionCall) otto.Value {
	return fetchFrom(FETCH_DISK, &call, &f)
}

// fetch.FromHtml
// ex: fetch.FromHtml("GET", "/path", "#selector_id")
func (f JSFetch) FromHtml(call otto.FunctionCall) otto.Value {
	return fetchFrom(FETCH_HTML, &call, &f)
}
