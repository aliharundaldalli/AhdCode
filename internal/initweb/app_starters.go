package initweb

func appStarterDirs(options Options) []string {
	dirs := []string{
		"database",
		"storage",
		"storage/private",
		"Views",
		"Views/Layouts",
		"Views/Members",
		"Views/Admin",
		"Views/Admin/Users",
	}
	if options.Starter == StarterMVC {
		dirs = append(dirs, "Routes", "Controllers", "Models")
	}
	return dirs
}

func appStarterFiles(options Options) []fileSpec {
	files := []fileSpec{
		{relPath: "Config/Database.ahd", perm: 0o644, content: []byte(renderDatabaseConfig(options))},
		{relPath: "database/schema.sql", perm: 0o644, content: []byte(schemaSQL(options))},
		{relPath: "Components/Navbar.ahd", perm: 0o644, content: []byte(appNavbarSource)},
		{relPath: "Components/Footer.ahd", perm: 0o644, content: []byte(renderAppFooter(options))},
		{relPath: "Views/Layouts/Main.ahd", perm: 0o644, content: []byte(appMainLayoutSource)},
		{relPath: "Views/Layouts/Guest.ahd", perm: 0o644, content: []byte(appGuestLayoutSource)},
		{relPath: "Views/Layouts/Signed.ahd", perm: 0o644, content: []byte(appSignedLayoutSource)},
		{relPath: "Views/Home.ahd", perm: 0o644, content: []byte(appHomeViewSource)},
		{relPath: "Views/Login.ahd", perm: 0o644, content: []byte(appLoginViewSource)},
		{relPath: "Views/Members/Show.ahd", perm: 0o644, content: []byte(appMemberShowViewSource)},
		{relPath: "Views/Members/Edit.ahd", perm: 0o644, content: []byte(appMemberEditViewSource)},
		{relPath: "Views/Admin/Users/Index.ahd", perm: 0o644, content: []byte(appAdminIndexViewSource)},
		{relPath: "Views/Admin/Users/Show.ahd", perm: 0o644, content: []byte(appAdminShowViewSource)},
		{relPath: "Views/Admin/Users/Form.ahd", perm: 0o644, content: []byte(appAdminFormViewSource)},
		{relPath: "storage/private/welcome.txt", perm: 0o644, content: []byte("This file is private. AhdCode serves it only after a signed-in check.\n")},
	}
	if options.Starter == StarterMVC {
		files = append(files,
			fileSpec{relPath: "Routes/Web.ahd", perm: 0o644, content: []byte(mvcRoutesSource)},
			fileSpec{relPath: "Models/User.ahd", perm: 0o644, content: []byte(renderAppUserModel(options))},
			fileSpec{relPath: "Controllers/Home.ahd", perm: 0o644, content: []byte(mvcHomeControllerSource)},
			fileSpec{relPath: "Controllers/Auth.ahd", perm: 0o644, content: []byte(mvcAuthControllerSource)},
			fileSpec{relPath: "Controllers/Members.ahd", perm: 0o644, content: []byte(mvcMembersControllerSource)},
			fileSpec{relPath: "Controllers/AdminUsers.ahd", perm: 0o644, content: []byte(mvcAdminControllerSource)},
		)
		return files
	}
	files = append(files,
		fileSpec{relPath: "routes.ahd", perm: 0o644, content: []byte(crudRoutesSource)},
		fileSpec{relPath: "auth.ahd", perm: 0o644, content: []byte(crudAuthSource)},
		fileSpec{relPath: "users.ahd", perm: 0o644, content: []byte(renderCRUDUsers(options))},
	)
	return files
}

func renderAppFooter(options Options) string {
	return `bring Web
from Web bring HTMLNode

footer: Function := (siteName: String) -> HTMLNode {
    return Web.UI.footer(
        [
            Web.UI.div(
                [
                    Web.UI.p(siteName, {"class": "app-footer-name"})
                    Web.UI.p("` + options.footerCredit() + `", {"class": "app-footer-meta"})
                ]
                {"class": "app-footer-inner"}
            )
        ]
        {"class": "app-footer"}
    )
}
`
}

func renderAppUserModel(options Options) string {
	if options.isMySQL() {
		return appMySQLUserSource
	}
	return appSQLiteUserSource
}

func renderCRUDUsers(options Options) string {
	return renderAppUserModel(options) + crudUserHandlersSource
}

const mvcAppSource = `require("Config/App.ahd")
require("Config/Mail.ahd")
require("Config/Database.ahd")
require("Routes/Web.ahd")
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
site.managedAssets("/assets", "public")
registerRoutes(routes)
write("{application.name} — http://{application.address()}")
site.start()
`

const crudAppSource = `require("Config/App.ahd")
require("Config/Mail.ahd")
require("Config/Database.ahd")
require("routes.ahd")
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
site.managedAssets("/assets", "public")
registerRoutes(routes)
write("{application.name} — http://{application.address()}")
site.start()
`

const mvcRoutesSource = `require("Controllers/Home.ahd")
require("Controllers/Auth.ahd")
require("Controllers/Members.ahd")
require("Controllers/AdminUsers.ahd")
bring Web
from Web bring RouteSet

registerRoutes: Function := (routes: RouteSet) -> Nothing {
    routes.get("/", home)
    routes.get("/login", login)
    routes.post("/login", loginSubmit)
    routes.post("/logout", logout)
    routes.get("/members/*", memberShow, signedIn)
    routes.get("/settings", memberEdit, signedIn)
    routes.post("/settings", memberUpdate, signedIn)
    routes.get("/private/welcome", privateWelcome, signedIn)
    routes.get("/admin/users", adminUsersIndex, signedIn, administrator)
    routes.get("/admin/users/new", adminUsersNew, signedIn, administrator)
    routes.post("/admin/users", adminUsersCreate, signedIn, administrator)
    routes.get("/admin/users/*", adminUsersShow, signedIn, administrator)
    routes.post("/admin/users/update", adminUsersUpdate, signedIn, administrator)
    routes.post("/admin/users/delete", adminUsersDelete, signedIn, administrator)
}
`

const crudRoutesSource = `require("auth.ahd")
require("users.ahd")
require("Views/Home.ahd")
bring Web
from Web bring (RouteSet, RequestContext, Response)

registerRoutes: Function := (routes: RouteSet) -> Nothing {
    routes.get("/", home)
    routes.get("/login", login)
    routes.post("/login", loginSubmit)
    routes.post("/logout", logout)
    routes.get("/members/*", memberShow, signedIn)
    routes.get("/settings", memberEdit, signedIn)
    routes.post("/settings", memberUpdate, signedIn)
    routes.get("/private/welcome", privateWelcome, signedIn)
    routes.get("/admin/users", adminUsersIndex, signedIn, administrator)
    routes.get("/admin/users/new", adminUsersNew, signedIn, administrator)
    routes.post("/admin/users", adminUsersCreate, signedIn, administrator)
    routes.get("/admin/users/*", adminUsersShow, signedIn, administrator)
    routes.post("/admin/users/update", adminUsersUpdate, signedIn, administrator)
    routes.post("/admin/users/delete", adminUsersDelete, signedIn, administrator)
}

home: Function := (context: RequestContext) -> Response {
    return context.respond(homeView(configuration(), usersPublicRows()))
}
`

const appNavbarSource = `bring Web
from Web bring (HTMLNode, RequestContext)

brandMark: Function := (href: String, siteName: String) -> HTMLNode {
    return Web.UI.a(href, siteName, {"class": "app-brand"})
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
                            Web.UI.a("/login", "Log In", {"class": "app-nav-link"})
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
    role: String
    context: RequestContext
) -> HTMLNode {
    links: Local List<HTMLNode> := [
        Web.UI.a("/", "Home", {"class": "app-nav-link"})
        Web.UI.a("/settings", "Profile", {"class": "app-nav-link"})
    ]
    if role == "administrator" {
        links.add(Web.UI.a("/admin/users", "Users", {"class": "app-nav-link"}))
    }
    links.add(Web.UI.span(userName, {"class": "app-nav-user"}))
    links.add(
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
    )
    return Web.UI.nav(
        [
            Web.UI.div(
                [
                    brandMark("/", siteName)
                    Web.UI.div(links, {"class": "app-nav-links"})
                ]
                {"class": "app-topbar"}
            )
        ]
        {"class": "app-navbar"}
    )
}
`

const appMainLayoutSource = `require("Components/Navbar.ahd")
require("Components/Footer.ahd")
bring Web
from Web bring (HTMLNode, Response, AppConfig)

starterAssets: Function := () -> List<HTMLNode> {
    return [
        Web.Assets.stylesheet("vendor/bootstrap/bootstrap.min.css")
        Web.Assets.stylesheet("style.css")
        Web.Assets.script("vendor/bootstrap/bootstrap.bundle.min.js")
        Web.Assets.script("main.js")
    ]
}

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
        starterAssets()
        status
    )
}
`

const appGuestLayoutSource = `require("Views/Layouts/Main.ahd")
bring Web
from Web bring (HTMLNode, Response, AppConfig)

guestLayout: Function := (
    config: AppConfig
    title: String
    content: List<HTMLNode>
    status: Int := 200
) -> Response {
    return mainLayout(config, title, content, status)
}
`

const appSignedLayoutSource = `require("Components/Navbar.ahd")
require("Components/Footer.ahd")
require("Views/Layouts/Main.ahd")
bring Web
from Web bring (HTMLNode, Response, AppConfig, RequestContext)

signedLayout: Function := (
    context: RequestContext
    config: AppConfig
    title: String
    userName: String
    role: String
    content: List<HTMLNode>
    status: Int := 200
) -> Response {
    return Web.page(
        "{title} — {config.name}"
        [
            Web.UI.div(
                [
                    appNavbar(config.name, userName, role, context)
                    Web.UI.main(content, {"class": "app-main app-dashboard"})
                    footer(config.name)
                ]
                {"class": "app-shell"}
            )
        ]
        starterAssets()
        status
    )
}
`

const appHomeViewSource = `require("Views/Layouts/Main.ahd")
bring Web
from Web bring (HTMLNode, Response, AppConfig)

homeView: Function := (
    config: AppConfig
    members: List<HTMLNode>
) -> Response {
    return mainLayout(
        config
        "Welcome"
        [
            Web.UI.section(
                [
                    Web.UI.h1("Welcome to {config.name}")
                    Web.UI.p(
                        "A public home page, member profiles, and an administrator user list."
                        {"class": "app-lead"}
                    )
                    Web.UI.div(
                        [Web.UI.a("/login", "Log In", {"class": "app-button"})]
                        {"class": "app-actions"}
                    )
                ]
                {"class": "app-stage"}
            )
            Web.UI.section(
                [
                    Web.UI.h2("Members")
                    Web.UI.table(
                        [
                            Web.UI.thead(
                                [
                                    Web.UI.tr(
                                        [Web.UI.th("Name"), Web.UI.th("Role")]
                                    )
                                ]
                            )
                            Web.UI.tbody(members)
                        ]
                        {"class": "table"}
                    )
                ]
                {"class": "app-notes"}
            )
        ]
    )
}
`

const appLoginViewSource = `require("Views/Layouts/Guest.ahd")
bring Web
from Web bring (
    RequestContext
    Response
    OldInput
    ValidationErrors
    HTMLNode
)

loginView: Function := (
    context: RequestContext
    old: OldInput
    errors: ValidationErrors
    message: String
    status: Int := 200
) -> Response {
    return guestLayout(
        configuration()
        "Log In"
        [
            Web.UI.div(
                [
                    loginAlerts(context, errors, message)
                    Web.UI.h1("Log In", {"class": "h3 mb-3"})
                    Web.UI.formTo(
                        "/login"
                        "post"
                        [
                            Web.UI.csrfField(context)
                            Web.UI.labelFor("email", "Email")
                            Web.UI.input(
                                "email"
                                "email"
                                old.value("email")
                                {
                                    "id": "email"
                                    "class": "form-control"
                                    "autocomplete": "username"
                                }
                            )
                            Web.UI.labelFor("password", "Password")
                            Web.UI.input(
                                "password"
                                "password"
                                ""
                                {
                                    "id": "password"
                                    "class": "form-control"
                                    "autocomplete": "current-password"
                                }
                            )
                            Web.UI.button(
                                "Log In"
                                {
                                    "type": "submit"
                                    "class": "app-button w-100 mt-3"
                                }
                            )
                        ]
                    )
                    Web.UI.pNodes(
                        [Web.UI.a("/", "Back to home")]
                        {"class": "mt-3 mb-0"}
                    )
                ]
                {"class": "app-login-card"}
            )
        ]
        status
    )
}

loginAlerts: Function := (
    context: RequestContext
    errors: ValidationErrors
    message: String
) -> HTMLNode {
    items: Local List<HTMLNode> := []
    notice: Local String? := context.flashTake("notice")
    if notice != null {
        items.add(Web.UI.p(notice, {"class": "app-flash"}))
    }
    if message != "" {
        items.add(Web.UI.p(message, {"class": "app-flash", "role": "alert"}))
    }
    for item in errors.messages() {
        items.add(Web.UI.p(item, {"class": "app-flash", "role": "alert"}))
    }
    return Web.UI.div(items)
}
`

const appMemberShowViewSource = `require("Views/Layouts/Signed.ahd")
bring Web
from Web bring (RequestContext, Response, HTMLNode, AppConfig)

memberShowView: Function := (
    context: RequestContext
    config: AppConfig
    userName: String
    role: String
    shownName: String
    shownRole: String
    publicId: String
    canEdit: Bool
    status: Int := 200
) -> Response {
    actions: Local List<HTMLNode> := []
    if canEdit {
        actions.add(Web.UI.a("/settings", "Edit profile", {"class": "app-button"}))
    }
    return signedLayout(
        context
        config
        shownName
        userName
        role
        [
            Web.UI.h1(shownName)
            Web.UI.p(roleLabel(shownRole))
            Web.UI.p("Public id: {publicId}")
            Web.UI.div(actions, {"class": "app-actions"})
        ]
        status
    )
}
`

const appMemberEditViewSource = `require("Views/Layouts/Signed.ahd")
bring Web
from Web bring (
    RequestContext
    Response
    ValidationErrors
    HTMLNode
    AppConfig
)

memberEditView: Function := (
    context: RequestContext
    config: AppConfig
    userName: String
    role: String
    name: String
    errors: ValidationErrors
    status: Int := 200
) -> Response {
    items: Local List<HTMLNode> := []
    for item in errors.messages() {
        items.add(Web.UI.p(item, {"class": "app-flash", "role": "alert"}))
    }
    return signedLayout(
        context
        config
        "Profile"
        userName
        role
        [
            Web.UI.div(items)
            Web.UI.h1("Your profile")
            Web.UI.formTo(
                "/settings"
                "post"
                [
                    Web.UI.csrfField(context)
                    Web.UI.labelFor("name", "Name")
                    Web.UI.input(
                        "text"
                        "name"
                        name
                        {"id": "name", "class": "form-control"}
                    )
                    Web.UI.button(
                        "Save"
                        {"type": "submit", "class": "app-button mt-3"}
                    )
                ]
            )
        ]
        status
    )
}
`

const appAdminIndexViewSource = `require("Views/Layouts/Signed.ahd")
bring Web
from Web bring (RequestContext, Response, HTMLNode, AppConfig)

adminUsersIndexView: Function := (
    context: RequestContext
    config: AppConfig
    userName: String
    role: String
    rows: List<HTMLNode>
    notice: String
) -> Response {
    flash: Local List<HTMLNode> := []
    if notice != "" {
        flash.add(Web.UI.p(notice, {"class": "app-flash", "role": "status"}))
    }
    return signedLayout(
        context
        config
        "Users"
        userName
        role
        [
            Web.UI.div(flash)
            Web.UI.h1("Users")
            Web.UI.pNodes(
                [Web.UI.a("/admin/users/new", "Create user", {"class": "app-button"})]
            )
            Web.UI.table(
                [
                    Web.UI.thead(
                        [
                            Web.UI.tr(
                                [
                                    Web.UI.th("Name")
                                    Web.UI.th("Email")
                                    Web.UI.th("Role")
                                    Web.UI.th("Public id")
                                ]
                            )
                        ]
                    )
                    Web.UI.tbody(rows)
                ]
                {"class": "table"}
            )
        ]
    )
}
`

const appAdminShowViewSource = `require("Views/Layouts/Signed.ahd")
bring Web
from Web bring (RequestContext, Response, HTMLNode, AppConfig)

adminUserShowView: Function := (
    context: RequestContext
    config: AppConfig
    userName: String
    role: String
    shownName: String
    shownEmail: String
    shownRole: String
    publicId: String
) -> Response {
    return signedLayout(
        context
        config
        shownName
        userName
        role
        [
            Web.UI.h1(shownName)
            Web.UI.p(shownEmail)
            Web.UI.p(roleLabel(shownRole))
            Web.UI.p("Public id: {publicId}")
            Web.UI.formTo(
                "/admin/users/delete"
                "post"
                [
                    Web.UI.csrfField(context)
                    Web.UI.input(
                        "hidden"
                        "public_id"
                        publicId
                        {}
                    )
                    Web.UI.button(
                        "Delete"
                        {"type": "submit", "class": "app-button"}
                    )
                ]
            )
        ]
    )
}
`

const appAdminFormViewSource = `require("Views/Layouts/Signed.ahd")
bring Web
from Web bring (
    RequestContext
    Response
    OldInput
    ValidationErrors
    HTMLNode
    AppConfig
)

adminUserFormView: Function := (
    context: RequestContext
    config: AppConfig
    userName: String
    role: String
    old: OldInput
    errors: ValidationErrors
    status: Int := 200
) -> Response {
    items: Local List<HTMLNode> := []
    for item in errors.messages() {
        items.add(Web.UI.p(item, {"class": "app-flash", "role": "alert"}))
    }
    return signedLayout(
        context
        config
        "Create user"
        userName
        role
        [
            Web.UI.div(items)
            Web.UI.h1("Create user")
            Web.UI.formTo(
                "/admin/users"
                "post"
                [
                    Web.UI.csrfField(context)
                    Web.UI.labelFor("name", "Name")
                    Web.UI.input(
                        "text"
                        "name"
                        old.value("name")
                        {"id": "name", "class": "form-control"}
                    )
                    Web.UI.labelFor("email", "Email")
                    Web.UI.input(
                        "email"
                        "email"
                        old.value("email")
                        {"id": "email", "class": "form-control"}
                    )
                    Web.UI.labelFor("password", "Password")
                    Web.UI.input(
                        "password"
                        "password"
                        ""
                        {"id": "password", "class": "form-control"}
                    )
                    Web.UI.labelFor("role", "Role")
                    Web.UI.input(
                        "text"
                        "role"
                        old.value("role")
                        {"id": "role", "class": "form-control"}
                    )
                    Web.UI.button(
                        "Create"
                        {"type": "submit", "class": "app-button mt-3"}
                    )
                ]
            )
        ]
        status
    )
}
`
