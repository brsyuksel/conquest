conquest
  .Host("https://10.0.2.2")
  .Headers({
  	"X-Conquest": "v0.1.0"
  })
  .ConquestHeaders()
  .Users(1, function(users){
    users
      .Every(function(user){
        /*
        * first reach root path
        * for getting _xsrf cookie
        */
        user
          .Do("GET", "/")
          .Response
            .StatusCode(200)
            .Header("Server", "TornadoServer/4.1")
            .Header("Content-Length", "12")
        ;

        /*
        * reach the /static page for obtaining
        * caching header "Etag"
        */
        user
          .Do("GET", "/static")
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
        * skipped transaction
        */
        user
          .Do("GET", "/")
          .Skip()
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
            "file": function(fetch){ return fetch.FromDisk("test_files/test.md"); },
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
            "file": function(fetch){ return fetch.FromDisk("test_files/test.png"); },
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
          .Body({
            "_xsrf" : function(fetch){ return fetch.FromCookie("_xsrf"); }
          })
          .Response
            .StatusCode(200)
        ;
      })
  });