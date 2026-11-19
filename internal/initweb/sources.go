package initweb

func renderApp(options Options) string {
	switch options.Starter {
	case StarterBasic:
		return basicAppSource
	case StarterAdmin:
		return adminAppSource
	case StarterMVC:
		return mvcAppSource
	case StarterCRUD:
		return crudAppSource
	default:
		return emptyAppSource
	}
}

func renderHome(options Options) string {
	switch options.Starter {
	case StarterBasic:
		return basicHomeSource
	case StarterAdmin:
		return adminHomeSource
	default:
		return emptyHomeSource
	}
}

func renderNavbar(options Options) string {
	if options.Starter == StarterAdmin {
		return adminNavbarSource
	}
	return publicNavbarSource
}

func renderMainLayout(options Options) string {
	if options.Starter == StarterAdmin {
		return adminMainLayoutSource
	}
	return publicMainLayoutSource
}

func renderDatabaseConfig(options Options) string {
	if options.isMySQL() {
		return mysqlDatabaseSource
	}
	return sqliteDatabaseSource
}

func renderUsersRepo(options Options) string {
	if options.isMySQL() {
		return mysqlUsersSource
	}
	return sqliteUsersSource
}

func schemaSQL(options Options) string {
	if options.isAppStarter() {
		if options.isMySQL() {
			return appMySQLSchemaSQL
		}
		return appSQLiteSchemaSQL
	}
	if options.isMySQL() {
		return mysqlSchemaSQL
	}
	return sqliteSchemaSQL
}

const emptyAppSource = `require("Config/App.ahd")
require("Pages/Home.ahd")
bring Web
from Web bring (App, SessionStore)

site: App := Web.app(application)
sessions: SessionStore := Web.sessions(
    "ahdsession"
    86400
    application.isSecure()
    "Lax"
)
routes := Web.routes(site, sessions)
site.assets("/assets", "public")
routes.get("/", home)
write("{application.name} — http://{application.address()}")
site.start()
`

const basicAppSource = `require("Config/App.ahd")
require("Config/Mail.ahd")
require("Pages/Home.ahd")
require("Pages/About.ahd")
require("Pages/Errors.ahd")
bring Web
from Web bring (App, SessionStore)

site: App := Web.app(application)
sessions: SessionStore := Web.sessions(
    "ahdsession"
    86400
    application.isSecure()
    "Lax"
)
routes := Web.routes(site, sessions)
site.assets("/assets", "public")
routes.get("/", home)
routes.get("/hakkinda", about)
routes.get("/ornek-hata", exampleError)
site.notFound(notFound)
site.internalError(serverError)
write("{application.name} — http://{application.address()}")
site.start()
`

const adminAppSource = `require("Config/App.ahd")
require("Config/Mail.ahd")
require("Config/Database.ahd")
require("Pages/Home.ahd")
require("Pages/Login.ahd")
require("Pages/Dashboard.ahd")
require("Services/Auth.ahd")
bring Web
from Web bring (App, SessionStore)

site: App := Web.app(application)
sessions: SessionStore := Web.sessions(
    "ahdsession"
    86400
    application.isSecure()
    "Lax"
)
routes := Web.routes(site, sessions)
site.assets("/assets", "public")
routes.get("/", home)
routes.get("/login", login)
routes.post("/login", loginSubmit)
routes.post("/logout", logout)
routes.get("/dashboard", dashboard, signedIn)
write("{application.name} — http://{application.address()}")
site.start()
`

const publicNavbarSource = `bring Web
from Web bring HTMLNode

brandMark: Function := (href: String, siteName: String) -> HTMLNode {
    return Web.UI.aNodes(
        href
        [
            Web.UI.img(
                "/assets/ahdcode-logo.png"
                "AhdCode"
                {"class": "app-logo"}
            )
            Web.UI.span(siteName, {"class": "app-brand-name"})
        ]
        {"class": "app-brand"}
    )
}

navbar: Function := (siteName: String) -> HTMLNode {
    return Web.UI.nav(
        [
            Web.UI.div(
                [
                    brandMark("/", siteName)
                    Web.UI.div(
                        [Web.UI.a("/", "Home", {"class": "app-nav-link"})]
                        {"class": "app-nav-links"}
                    )
                ]
                {"class": "app-topbar"}
            )
        ]
        {"class": "app-navbar"}
    )
}
`

const adminNavbarSource = `bring Web
from Web bring (HTMLNode, RequestContext)

brandMark: Function := (href: String, siteName: String) -> HTMLNode {
    return Web.UI.aNodes(
        href
        [
            Web.UI.img(
                "/assets/ahdcode-logo.png"
                "AhdCode"
                {"class": "app-logo"}
            )
            Web.UI.span(siteName, {"class": "app-brand-name"})
        ]
        {"class": "app-brand"}
    )
}

navbar: Function := (siteName: String) -> HTMLNode {
    return Web.UI.nav(
        [
            Web.UI.div(
                [
                    brandMark("/", siteName)
                    Web.UI.div(
                        [
                            Web.UI.a("/", "Home", {"class": "app-nav-link"})
                            Web.UI.a(
                                "/login"
                                "Log in"
                                {"class": "app-nav-link"}
                            )
                        ]
                        {"class": "app-nav-links"}
                    )
                ]
                {"class": "app-topbar"}
            )
        ]
        {"class": "app-navbar"}
    )
}

appNavbar: Function := (
    siteName: String
    userName: String
    context: RequestContext
) -> HTMLNode {
    return Web.UI.nav(
        [
            Web.UI.div(
                [
                    brandMark("/dashboard", siteName)
                    Web.UI.div(
                        [
                            Web.UI.span(userName, {"class": "app-nav-user"})
                            Web.UI.formTo(
                                "/logout"
                                "post"
                                [
                                    Web.UI.csrfField(context)
                                    Web.UI.button(
                                        "Log out"
                                        {
                                            "type": "submit"
                                            "class": "app-nav-link app-nav-button"
                                        }
                                    )
                                ]
                                {"class": "app-logout"}
                            )
                        ]
                        {"class": "app-nav-links"}
                    )
                ]
                {"class": "app-topbar"}
            )
        ]
        {"class": "app-navbar"}
    )
}
`

const publicMainLayoutSource = `require("Components/Navbar.ahd")
require("Components/Footer.ahd")
bring Web
from Web bring (HTMLNode, Response, AppConfig)

mainLayout: Function := (
    config: AppConfig
    title: String
    content: List<HTMLNode>
    status: Int := 200
) -> Response {
    return Web.page(
        "{title} — {config.name}"
        [
            Web.UI.div(
                [
                    navbar(config.name)
                    Web.UI.main(content, {"class": "app-main"})
                    footer(config.name)
                ]
                {"class": "app-shell"}
            )
        ]
        [
            Web.UI.stylesheet("/assets/vendor/bootstrap/bootstrap.min.css")
            Web.UI.stylesheet("/assets/style.css")
            Web.UI.element(
                "script"
                {"src": "/assets/vendor/bootstrap/bootstrap.bundle.min.js"}
                []
            )
            Web.UI.element("script", {"src": "/assets/main.js"}, [])
        ]
        status
    )
}
`

const adminMainLayoutSource = `require("Components/Navbar.ahd")
require("Components/Footer.ahd")
bring Web
from Web bring (HTMLNode, Response, AppConfig, RequestContext)

mainLayout: Function := (
    config: AppConfig
    title: String
    content: List<HTMLNode>
    status: Int := 200
) -> Response {
    return Web.page(
        "{title} — {config.name}"
        [
            Web.UI.div(
                [
                    navbar(config.name)
                    Web.UI.main(content, {"class": "app-main"})
                    footer(config.name)
                ]
                {"class": "app-shell"}
            )
        ]
        starterHead()
        status
    )
}

appLayout: Function := (
    context: RequestContext
    config: AppConfig
    title: String
    userName: String
    content: List<HTMLNode>
    status: Int := 200
) -> Response {
    return Web.page(
        "{title} — {config.name}"
        [
            Web.UI.div(
                [
                    appNavbar(config.name, userName, context)
                    Web.UI.main(content, {"class": "app-main app-dashboard"})
                    footer(config.name)
                ]
                {"class": "app-shell"}
            )
        ]
        starterHead()
        status
    )
}

starterHead: Function := () -> List<HTMLNode> {
    return [
        Web.UI.stylesheet("/assets/vendor/bootstrap/bootstrap.min.css")
        Web.UI.stylesheet("/assets/style.css")
        Web.UI.element(
            "script"
            {"src": "/assets/vendor/bootstrap/bootstrap.bundle.min.js"}
            []
        )
        Web.UI.element("script", {"src": "/assets/main.js"}, [])
    ]
}
`

const emptyHomeSource = `require("Config/App.ahd")
require("Layouts/Main.ahd")
bring Web
from Web bring (RequestContext, Response, AppConfig)

home: Function := (context: RequestContext) -> Response {
    config: Local AppConfig := configuration()
    return context.respond(
        mainLayout(
            config
            "Welcome"
            [
                Web.UI.section(
                    [
                        Web.UI.img(
                            "/assets/ahdcode-logo.png"
                            "AhdCode"
                            {"class": "app-hero-logo"}
                        )
                        Web.UI.h1("Let's get started")
                        Web.UI.p(
                            "Your application is ready."
                            {"class": "app-lead"}
                        )
                        Web.UI.p(
                            "Build fast. Stay explicit. Keep control."
                            {"class": "app-sub"}
                        )
                        Web.UI.div(
                            [
                                Web.UI.a(
                                    "#files"
                                    "Get started"
                                    {"class": "app-button"}
                                )
                            ]
                            {"class": "app-actions"}
                        )
                    ]
                    {"class": "app-stage"}
                )
                Web.UI.section(
                    [
                        Web.UI.h2("Start here")
                        Web.UI.ul(
                            [
                                Web.UI.li("Pages/Home.ahd")
                                Web.UI.li("Layouts/Main.ahd")
                                Web.UI.li("Components/Navbar.ahd")
                                Web.UI.li("public/style.css")
                                Web.UI.li("public/main.js")
                            ]
                            {"class": "app-files"}
                        )
                    ]
                    {"id": "files", "class": "app-notes"}
                )
            ]
        )
    )
}
`

const basicSiteSource = `bring Templates
from Templates bring TemplateSet

pages: TemplateSet := Templates.files("templates")
`

const basicAboutSource = `require("Config/App.ahd")
require("Pages/Site.ahd")
bring Templates
from Templates bring TemplateSet
bring Web
from Web bring (RequestContext, Response, AppConfig)

about: Function := (context: RequestContext) -> Response {
    pages: Global TemplateSet
    config: Local AppConfig := configuration()
    return context.respond(Web.markup(pages.layout(
        "layouts/site.html"
        "pages/about.html"
        Templates.object({
            "title": Templates.fromString("Hakkında")
            "name": Templates.fromString(config.name)
            "note": Templates.fromString("Bu sayfa aynı yerleşimi kullanır.")
        })
    )))
}
`

const basicErrorsSource = `require("Config/App.ahd")
require("Pages/Site.ahd")
bring Templates
from Templates bring TemplateSet
bring Web
from Web bring (Request, RequestContext, Response, AppConfig)

notFound: Function := (request: Request) -> Response {
    pages: Global TemplateSet
    config: Local AppConfig := configuration()
    return Web.markup(
        pages.layout(
            "layouts/site.html"
            "pages/not-found.html"
            Templates.object({
                "title": Templates.fromString("Sayfa bulunamadı")
                "name": Templates.fromString(config.name)
            })
        )
        404
    )
}

serverError: Function := (request: Request) -> Response {
    pages: Global TemplateSet
    config: Local AppConfig := configuration()
    return Web.markup(
        pages.layout(
            "layouts/site.html"
            "pages/server-error.html"
            Templates.object({
                "title": Templates.fromString("Sunucu hatası")
                "name": Templates.fromString(config.name)
            })
        )
        500
    )
}

exampleError: Function := (context: RequestContext) -> Response {
    toss(DomainError("ornek"))
}
`

const basicHomeSource = `require("Config/App.ahd")
require("Pages/Site.ahd")
bring Templates
from Templates bring TemplateSet
bring Web
from Web bring (RequestContext, Response, AppConfig)

home: Function := (context: RequestContext) -> Response {
    pages: Global TemplateSet
    config: Local AppConfig := configuration()
    return context.respond(Web.markup(pages.layout(
        "layouts/site.html"
        "pages/home.html"
        Templates.object({
            "title": Templates.fromString("Ana sayfa")
            "name": Templates.fromString(config.name)
            "note": Templates.fromString("İstanbul'da küçük bir başlangıç. Ortam: {config.environment}.")
            "topics": Templates.list([
                Templates.fromString("Şablonlar")
                Templates.fromString("Yerleşim")
                Templates.fromString("Hata sayfaları")
            ])
        })
    )))
}
`

const adminHomeSource = `require("Config/App.ahd")
require("Layouts/Main.ahd")
bring Web
from Web bring (RequestContext, Response, AppConfig)

home: Function := (context: RequestContext) -> Response {
    config: Local AppConfig := configuration()
    return context.respond(
        mainLayout(
            config
            "Welcome"
            [
                Web.UI.section(
                    [
                        Web.UI.img(
                            "/assets/ahdcode-logo.png"
                            "AhdCode"
                            {"class": "app-hero-logo"}
                        )
                        Web.UI.h1("Let's get started")
                        Web.UI.p(
                            "A public home page and a signed-in dashboard."
                            {"class": "app-lead"}
                        )
                        Web.UI.p(
                            "Log in to open the administrator dashboard."
                            {"class": "app-sub"}
                        )
                        Web.UI.div(
                            [
                                Web.UI.a(
                                    "/login"
                                    "Log in"
                                    {"class": "app-button"}
                                )
                            ]
                            {"class": "app-actions"}
                        )
                    ]
                    {"class": "app-stage"}
                )
            ]
        )
    )
}
`

const sqliteDatabaseSource = `// Database configuration.
//
// SQLite path and driver are read here. Pages never reach for Env.
bring Env
bring SQLite
from SQLite bring Database

database: Database := SQLite.open(Env.getOr("DB_DATABASE", "database/app.db"))

connection: Function := () -> Database {
    database: Global Database
    return database
}

databaseDriver: Function := () -> String {
    return Env.getOr("DB_DRIVER", "sqlite")
}
`

const mysqlDatabaseSource = `// Database configuration.
//
// Credentials are read here and nowhere else. Errors never include the
// password.
bring Env
bring MySQL
from MySQL bring MySQLDatabase

database: MySQLDatabase := MySQL.connect(
    Env.getOr("DB_HOST", "127.0.0.1")
    Env.getOr("DB_USERNAME", "")
    Env.getOr("DB_PASSWORD", "")
    databasePort()
    Env.getOr("DB_DATABASE", "")
    Env.getOr("DB_SECURITY", "tls")
)

connection: Function := () -> MySQLDatabase {
    database: Global MySQLDatabase
    return database
}

databaseDriver: Function := () -> String {
    return Env.getOr("DB_DRIVER", "mysql")
}

databasePort: Function := () -> Int {
    raw: Local String := Env.getOr("DB_PORT", "3306")
    value: Local Int := 3306
    attempt {
        value = int(raw)
    }
    except DomainError {
        return 3306
    }
    except OverflowError {
        return 3306
    }

    return value
}
`

const sqliteUsersSource = `require("Config/Database.ahd")
bring SQLite
from SQLite bring SQLiteValue

UserRecord: Class<> := {
    structure: Attributes := (
        id: Int
        name: String
        email: String
        passwordHash: String
        admin: Bool
    )
}

usersFindByEmail: Function := (email: String) -> UserRecord? {
    rows: Local List<Pair<String, SQLiteValue>> := connection(
        
    ).query(
        "SELECT id, name, email, password_hash, is_admin FROM users WHERE email = ? LIMIT 1"
        [SQLite.fromString(email)]
    )
    if len(rows) == 0 {
        return null
    }

    return usersFromRow(rows[0])
}

usersFromRow: Function := (row: Pair<String, SQLiteValue>) -> UserRecord {
    return UserRecord(
        id: row["id"].int()
        name: row["name"].string()
        email: row["email"].string()
        passwordHash: row["password_hash"].string()
        admin: row["is_admin"].int() == 1
    )
}
`

const mysqlUsersSource = `require("Config/Database.ahd")
bring MySQL
from MySQL bring MySQLValue

UserRecord: Class<> := {
    structure: Attributes := (
        id: Int
        name: String
        email: String
        passwordHash: String
        admin: Bool
    )
}

usersFindByEmail: Function := (email: String) -> UserRecord? {
    rows: Local List<Pair<String, MySQLValue>> := connection(
        
    ).query(
        "SELECT id, name, email, password_hash, is_admin FROM users WHERE email = ? LIMIT 1"
        [MySQL.fromString(email)]
    )
    if len(rows) == 0 {
        return null
    }

    return usersFromRow(rows[0])
}

usersFromRow: Function := (row: Pair<String, MySQLValue>) -> UserRecord {
    return UserRecord(
        id: row["id"].int()
        name: row["name"].string()
        email: row["email"].string()
        passwordHash: row["password_hash"].string()
        admin: row["is_admin"].int() == 1
    )
}
`

const sqliteSchemaSQL = `CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    is_admin INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
`

const mysqlSchemaSQL = `CREATE TABLE users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(80) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    is_admin TINYINT(1) NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`
