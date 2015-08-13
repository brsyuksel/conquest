__author__ = 'baris'

from tornado.web import Application, RequestHandler
from tornado.ioloop import IOLoop


class BaseHandler(RequestHandler):

    def initialize(self):
        self.set_header("Content-Type", "application/json")
        print(self.xsrf_token)

    def get_current_user(self):
        return self.get_secure_cookie("user")


class RootHandler(BaseHandler):

    def get(self, *args, **kwargs):
        u = self.current_user or bytes("", "utf-8")
        self.write({
            "user": u.decode("utf-8"),
        })
        self.finish()


class AuthenticationHandler(BaseHandler):

    def post(self, *args, **kwargs):
        u = self.get_argument("user", None)
        p = self.get_argument("pass", None)
        if u == "root" and p == "toor":
            self.set_secure_cookie("user", "root")
            self.set_status(200)
            self.write({
                "message": "ok"
            })
        else:
            self.set_status(401)
            self.write({
                "error": "invalid credentials",
                "message": "not ok"
            })

    def get(self, *args, **kwargs):
        if self.current_user:
            self.set_status(200)
            self.write({
                "message": "welcome back!",
                "meta": {"user": self.current_user.decode("utf-8")}
            })
        else:
            self.set_status(401)
            self.write({
                "error": "unauthorized",
                "message": "login first"
            })

    def delete(self, *args, **kwargs):
        if not self.current_user:
            self.set_status(403)
            self.finish()
        self.clear_all_cookies()
        self.set_status(200)


class ForbiddenHandler(BaseHandler):

    def get(self, *args, **kwargs):
        if not self.current_user:
            self.write({
                "error": "unauthorized",
                "message": "login first"
            })
            self.set_status(403)
            self.finish()

        self.write({
            "user": self.current_user.decode("utf-8")
        })
        self.set_status(200)


class QueryPostHandler(BaseHandler):

    def get(self, *args, **kwargs):
        q = self.get_query_argument("q", None)
        self.write({"q": q})

    def post(self, *args, **kwargs):
        q = self.get_argument("q", None)
        self.write({"q": q})


class FileUploadHandler(BaseHandler):

    def post(self, *args, **kwargs):
        f = self.request.files.get("file", None)
        if f is None:
            self.set_status(400)
            self.write({
                "error": "file"
            })
            self.finish()

        f = f[0]
        if f["content_type"] == "text/markdown":
            self.write({
                "name": f["filename"]
            })
        else:
            self.set_status(415)
            self.write({"error": f["content_type"]})


class HeadersHandler(BaseHandler):

    def get(self, *args, **kwargs):
        self.write({"asdasdasd": 1})
        self.set_etag_header()
        if self.check_etag_header():
            self._write_buffer = []
            self.set_status(304)
            return
        self.finish()


class MyApp(Application):

    def __init__(self):
        super(MyApp, self).__init__(handlers=[
            (r"/", RootHandler),
            (r"/auth", AuthenticationHandler),
            (r"/forbidden", ForbiddenHandler),
            (r"/searchlike", QueryPostHandler),
            (r"/static", HeadersHandler),
            (r"/file", FileUploadHandler),
        ], **{
            "cookie_secret": "sw28x*+&foyw3_&9*n*)tl(uj8m9az=udk+od&1x+o@q",
            "xsrf_cookies": True,
            "auto_reload": True
        })

MyApp().listen(2297)
IOLoop.instance().start()
