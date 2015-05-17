conquest
  .Host("http://10.0.2.2:2297")
  .Requests(100)
  .Headers({
  	"X-Conquest": "v0.1.0"
  })
  .Users(10, function(users){
    users
      .Every(function(user){
        /*
        * first reach root path
        */
        user
          .Do("GET", "/")
          .Response
            .StatusCode(200)
            .Header("Server", "TornadoServer/4.1")
            .Header("Content-Length", "16")
        ;

        /*
        * Try signing in with invalid credentials
        */

        user
          .Do("POST", "/auth")
          .Body({
            "user": "non-exists",
            "pass": "secret",
            "_xsrf": function(fetch){ return fetch.FromCookie("_xsrf"); },
          })
          .Response
            .StatusCode(401)
        ;

        /*
        * Sign in
        */

        user
          .Do("POST", "/auth")
          .Body({
            "user": "root",
            "pass": "toor",
            "_xsrf": function(fetch){ return fetch.FromCookie("_xsrf"); },
          })
        ;
      })
      .Then(function(user){
        /*
        * reach forbidden profile zone
        */
        user
          .Do("GET", "/forbidden")
          .Response
            .StatusCode(200)
        ;

        /*
        * try reach forbidden profile zone without cookies
        */
        user
          .Do("GET", "/forbidden")
          .ClearCookies()
          .Response
            .StatusCode(403)
        ;

        /*
        * Query via GET
        */
        user
          .Do("GET", "/searchlike")
          .Body({
            "q": "conquest"
          })
        ;

        /*
        * Query via POST
        */
        user
          .Do("POST", "/searchlike")
          .Body({
            "q": "conquest",
            "_xsrf": function(fetch){ return fetch.FromCookie("_xsrf"); }
          })
        ;

        /*
        * Fetch static file and its etag
        */
        user
          .Do("GET", "/static")
          .Response
            .StatusCode(200)
        ;
        
        /*
        * Check If-None-Match with etag
        */
        user
          .Do("GET", "/static")
          .SetHeader("If-None-Match", function(fetch){ return fetch.FromHeader("Etag"); })
          .Response
            .StatusCode(304)
        ;

        /*
        * Try upload a file
        */
        user
          .Do("POST", "/file")
          .Body({
            "file": function(fetch){ return fetch.FromDisk("test_files/", "text/markdown"); },
            "_xsrf": function(fetch){ return fetch.FromCookie("_xsrf"); }
          })
          .Response
            .StatusCode(200)
        ;

        /*
        * Try upload invalid mime type
        */
        user
          .Do("POST", "/file")
          .Body({
            "file": function(fetch){ return fetch.FromDisk("test_files/", "image/png"); },
            "_xsrf": function(fetch){ return fetch.FromCookie("_xsrf"); }
          })
          .Response
            .StatusCode(415)
        ;
      })
      .Finally(function(user){
        /*
        * Everybody should go home
        */
        user
          .Do("DELETE", "/auth")
          .Response
            .StatusCode(200)
        ;
      })
  });