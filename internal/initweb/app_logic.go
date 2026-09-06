package initweb

const appHelpersSource = `currentUserName: Function := (context: RequestContext) -> String {
    stored: Local String? := context.session.get("user_name")
    if stored != null {
        if stored != "" {
            return stored
        }
    }

    return "Member"
}

currentUserRole: Function := (context: RequestContext) -> String {
    stored: Local String? := context.session.get("user_role")
    if stored != null {
        if stored != "" {
            return stored
        }
    }

    return "member"
}

currentPublicID: Function := (context: RequestContext) -> String {
    stored: Local String? := context.session.get("user_public_id")
    if stored != null {
        return stored
    }

    return ""
}

pathPublicID: Function := (context: RequestContext) -> String {
    parts: Local List<String> := context.request.path().split("/")
    if len(parts) == 0 {
        return ""
    }

    return parts[len(parts) - 1]
}

roleLabel: Function := (role: String) -> String {
    if role == "administrator" {
        return "Administrator"
    }

    return "Member"
}

normalizeRole: Function := (role: String) -> String {
    value: Local String := role.trim().lower()
    if value == "administrator" {
        return "administrator"
    }

    return "member"
}

usersPublicRows: Function := () -> List<HTMLNode> {
    rows: Local List<HTMLNode> := []
    for user in usersAll() {
        rows.add(
            Web.UI.tr(
                [Web.UI.td(user.name), Web.UI.td(roleLabel(user.role))]
            )
        )
    }
    return rows
}
`

const appAuthCoreSource = `signedIn: Function := (context: RequestContext) -> Response? {
    if context.session.has("user_public_id") == false {
        return context.respond(Web.redirect("/login"))
    }

    return null
}

administrator: Function := (context: RequestContext) -> Response? {
    if context.session.has("user_public_id") == false {
        return context.respond(Web.redirect("/login"))
    }
    role: Local String? := context.session.get("user_role")
    if role == null {
        return context.respond(Web.text("Forbidden", 403))
    }
    if role != "administrator" {
        return context.respond(Web.text("Forbidden", 403))
    }

    return null
}

authenticate: Function := (
    context: RequestContext
    email: String
    password: String
) -> Bool {
    user: Local UserRecord? := usersFindByEmail(email)
    if user == null {
        return false
    }
    if Security.passwordVerify(password, user.passwordHash) == false {
        return false
    }
    context.session.rotate()
    context.session.set("user_public_id", user.publicId)
    context.session.set("user_name", user.name)
    context.session.set("user_role", user.role)
    return true
}

logout: Function := (context: RequestContext) -> Response {
    if context.csrfValid() == false {
        return context.respond(Web.text("Forbidden", 403))
    }
    context.session.destroy()
    return context.respond(Web.redirect("/"))
}

login: Function := (context: RequestContext) -> Response {
    return context.respond(
        loginView(context, context.form().old([]), Web.errors(), "")
    )
}

loginSubmit: Function := (context: RequestContext) -> Response {
    form: Local Form := context.form()
    if context.csrfValid() == false {
        return context.respond(Web.text("Forbidden", 403))
    }
    errors: Local ValidationErrors := Web.errors()
    errors.required("email", form.value("email"), "Email is required.")
    errors.email("email", form.value("email"), "Enter an email address.")
    errors.required("password", form.value("password"), "Password is required.")
    if errors.any() {
        return context.respond(
            loginView(context, form.old(["email"]), errors, "", 422)
        )
    }
    if authenticate(context, form.value("email"), form.value("password")) {
        context.flashSet("notice", "Signed in.")
        if currentUserRole(context) == "administrator" {
            return context.respond(Web.redirect("/admin/users"))
        }

        return context.respond(Web.redirect("/settings"))
    }

    return context.respond(
        loginView(
            context
            form.old(["email"])
            errors
            "Those credentials are not correct."
            422
        )
    )
}
`

const mvcHomeControllerSource = `require("Models/User.ahd")
require("Views/Home.ahd")
bring Web
from Web bring (RequestContext, Response, HTMLNode)

` + appHelpersSource + `

home: Function := (context: RequestContext) -> Response {
    return context.respond(homeView(configuration(), usersPublicRows()))
}
`

const mvcAuthControllerSource = `require("Models/User.ahd")
require("Views/Login.ahd")
require("Controllers/Home.ahd")
bring Security
bring Web
from Web bring (RequestContext, Response, Form, ValidationErrors)

` + appAuthCoreSource + `
`

const mvcMembersControllerSource = `require("Models/User.ahd")
require("Views/Members/Show.ahd")
require("Views/Members/Edit.ahd")
require("Controllers/Home.ahd")
require("Controllers/Auth.ahd")
bring HTTP
bring Web
from Web bring (RequestContext, Response, Form, OldInput, ValidationErrors)

memberShow: Function := (context: RequestContext) -> Response {
    publicId: Local String := pathPublicID(context)
    user: Local UserRecord? := usersFindByPublicID(publicId)
    if user == null {
        return context.respond(Web.text("Not Found", 404))
    }
    selfId: Local String := currentPublicID(context)
    role: Local String := currentUserRole(context)
    if role != "administrator" {
        if user.publicId != selfId {
            return context.respond(Web.text("Forbidden", 403))
        }
    }
    return context.respond(
        memberShowView(
            context
            configuration()
            currentUserName(context)
            role
            user.name
            user.role
            user.publicId
            user.publicId == selfId
        )
    )
}

memberEdit: Function := (context: RequestContext) -> Response {
    user: Local UserRecord? := usersFindByPublicID(currentPublicID(context))
    if user == null {
        return context.respond(Web.text("Not Found", 404))
    }
    return context.respond(
        memberEditView(
            context
            configuration()
            user.name
            user.role
            user.name
            Web.errors()
        )
    )
}

memberUpdate: Function := (context: RequestContext) -> Response {
    if context.csrfValid() == false {
        return context.respond(Web.text("Forbidden", 403))
    }
    form: Local Form := context.form()
    errors: Local ValidationErrors := Web.errors()
    errors.required("name", form.value("name"), "Name is required.")
    user: Local UserRecord? := usersFindByPublicID(currentPublicID(context))
    if user == null {
        return context.respond(Web.text("Not Found", 404))
    }
    if errors.any() {
        return context.respond(
            memberEditView(
                context
                configuration()
                user.name
                user.role
                form.value("name")
                errors
                422
            )
        )
    }
    usersUpdateName(user.publicId, form.value("name"))
    context.session.set("user_name", form.value("name"))
    return context.respond(Web.redirect("/members/{user.publicId}"))
}

privateWelcome: Function := (context: RequestContext) -> Response {
    return context.respond(HTTP.file("storage/private/welcome.txt", "text/plain"))
}
`

const mvcAdminControllerSource = `require("Models/User.ahd")
require("Views/Admin/Users/Index.ahd")
require("Views/Admin/Users/Show.ahd")
require("Views/Admin/Users/Form.ahd")
require("Controllers/Home.ahd")
require("Controllers/Auth.ahd")
bring Web
from Web bring (RequestContext, Response, Form, HTMLNode, ValidationErrors)

adminUsersIndex: Function := (context: RequestContext) -> Response {
    rows: Local List<HTMLNode> := []
    for user in usersAll() {
        rows.add(
            Web.UI.tr(
                [
                    Web.UI.tdNodes(
                        [Web.UI.a("/admin/users/{user.publicId}", user.name)]
                    )
                    Web.UI.td(user.email)
                    Web.UI.td(roleLabel(user.role))
                    Web.UI.td(user.publicId)
                ]
            )
        )
    }
    notice: Local String := ""
    flash: Local String? := context.flashTake("notice")
    if flash != null {
        notice = flash
    }
    return context.respond(
        adminUsersIndexView(
            context
            configuration()
            currentUserName(context)
            currentUserRole(context)
            rows
            notice
        )
    )
}

adminUsersNew: Function := (context: RequestContext) -> Response {
    return context.respond(
        adminUserFormView(
            context
            configuration()
            currentUserName(context)
            currentUserRole(context)
            context.form().old([])
            Web.errors()
        )
    )
}

adminUsersCreate: Function := (context: RequestContext) -> Response {
    if context.csrfValid() == false {
        return context.respond(Web.text("Forbidden", 403))
    }
    form: Local Form := context.form()
    errors: Local ValidationErrors := Web.errors()
    errors.required("name", form.value("name"), "Name is required.")
    errors.required("email", form.value("email"), "Email is required.")
    errors.email("email", form.value("email"), "Enter an email address.")
    errors.required("password", form.value("password"), "Password is required.")
    role: Local String := normalizeRole(form.value("role"))
    if usersEmailTaken(form.value("email")) {
        errors.required("email", "", "That email is already registered.")
    }
    if errors.any() {
        return context.respond(
            adminUserFormView(
                context
                configuration()
                currentUserName(context)
                currentUserRole(context)
                form.old(["name", "email", "role"])
                errors
                422
            )
        )
    }
    created: Local String := usersCreate(
        form.value("name")
        form.value("email")
        form.value("password")
        role
    )
    context.flashSet("notice", "User created.")
    return context.respond(Web.redirect("/admin/users/{created}"))
}

adminUsersShow: Function := (context: RequestContext) -> Response {
    user: Local UserRecord? := usersFindByPublicID(pathPublicID(context))
    if user == null {
        return context.respond(Web.text("Not Found", 404))
    }
    return context.respond(
        adminUserShowView(
            context
            configuration()
            currentUserName(context)
            currentUserRole(context)
            user.name
            user.email
            user.role
            user.publicId
        )
    )
}

adminUsersUpdate: Function := (context: RequestContext) -> Response {
    if context.csrfValid() == false {
        return context.respond(Web.text("Forbidden", 403))
    }
    form: Local Form := context.form()
    publicId: Local String := form.value("public_id")
    user: Local UserRecord? := usersFindByPublicID(publicId)
    if user == null {
        return context.respond(Web.text("Not Found", 404))
    }
    usersUpdateName(publicId, form.value("name"))
    return context.respond(Web.redirect("/admin/users/{publicId}"))
}

adminUsersDelete: Function := (context: RequestContext) -> Response {
    if context.csrfValid() == false {
        return context.respond(Web.text("Forbidden", 403))
    }
    if context.request.method() != "POST" {
        return context.respond(Web.text("Method Not Allowed", 405))
    }
    publicId: Local String := context.form().value("public_id")
    if publicId == currentPublicID(context) {
        context.flashSet("notice", "You cannot delete your own account.")
        return context.respond(Web.redirect("/admin/users"))
    }
    usersDelete(publicId)
    context.flashSet("notice", "User deleted.")
    return context.respond(Web.redirect("/admin/users"))
}
`

const crudAuthSource = `require("users.ahd")
require("Views/Login.ahd")
require("Views/Home.ahd")
bring Security
bring Web
from Web bring (RequestContext, Response, Form, ValidationErrors, HTMLNode)

` + appAuthCoreSource + `
`

const crudUserHandlersSource = `
require("Views/Members/Show.ahd")
require("Views/Members/Edit.ahd")
require("Views/Admin/Users/Index.ahd")
require("Views/Admin/Users/Show.ahd")
require("Views/Admin/Users/Form.ahd")
bring HTTP
bring Web
from Web bring (RequestContext, Response, Form, HTMLNode, ValidationErrors)

` + appHelpersSource + `

memberShow: Function := (context: RequestContext) -> Response {
    publicId: Local String := pathPublicID(context)
    user: Local UserRecord? := usersFindByPublicID(publicId)
    if user == null {
        return context.respond(Web.text("Not Found", 404))
    }
    selfId: Local String := currentPublicID(context)
    role: Local String := currentUserRole(context)
    if role != "administrator" {
        if user.publicId != selfId {
            return context.respond(Web.text("Forbidden", 403))
        }
    }
    return context.respond(
        memberShowView(
            context
            configuration()
            currentUserName(context)
            role
            user.name
            user.role
            user.publicId
            user.publicId == selfId
        )
    )
}

memberEdit: Function := (context: RequestContext) -> Response {
    user: Local UserRecord? := usersFindByPublicID(currentPublicID(context))
    if user == null {
        return context.respond(Web.text("Not Found", 404))
    }
    return context.respond(
        memberEditView(
            context
            configuration()
            user.name
            user.role
            user.name
            Web.errors()
        )
    )
}

memberUpdate: Function := (context: RequestContext) -> Response {
    if context.csrfValid() == false {
        return context.respond(Web.text("Forbidden", 403))
    }
    form: Local Form := context.form()
    errors: Local ValidationErrors := Web.errors()
    errors.required("name", form.value("name"), "Name is required.")
    user: Local UserRecord? := usersFindByPublicID(currentPublicID(context))
    if user == null {
        return context.respond(Web.text("Not Found", 404))
    }
    if errors.any() {
        return context.respond(
            memberEditView(
                context
                configuration()
                user.name
                user.role
                form.value("name")
                errors
                422
            )
        )
    }
    usersUpdateName(user.publicId, form.value("name"))
    context.session.set("user_name", form.value("name"))
    return context.respond(Web.redirect("/members/{user.publicId}"))
}

privateWelcome: Function := (context: RequestContext) -> Response {
    return context.respond(HTTP.file("storage/private/welcome.txt", "text/plain"))
}

adminUsersIndex: Function := (context: RequestContext) -> Response {
    rows: Local List<HTMLNode> := []
    for user in usersAll() {
        rows.add(
            Web.UI.tr(
                [
                    Web.UI.tdNodes(
                        [Web.UI.a("/admin/users/{user.publicId}", user.name)]
                    )
                    Web.UI.td(user.email)
                    Web.UI.td(roleLabel(user.role))
                    Web.UI.td(user.publicId)
                ]
            )
        )
    }
    notice: Local String := ""
    flash: Local String? := context.flashTake("notice")
    if flash != null {
        notice = flash
    }
    return context.respond(
        adminUsersIndexView(
            context
            configuration()
            currentUserName(context)
            currentUserRole(context)
            rows
            notice
        )
    )
}

adminUsersNew: Function := (context: RequestContext) -> Response {
    return context.respond(
        adminUserFormView(
            context
            configuration()
            currentUserName(context)
            currentUserRole(context)
            context.form().old([])
            Web.errors()
        )
    )
}

adminUsersCreate: Function := (context: RequestContext) -> Response {
    if context.csrfValid() == false {
        return context.respond(Web.text("Forbidden", 403))
    }
    form: Local Form := context.form()
    errors: Local ValidationErrors := Web.errors()
    errors.required("name", form.value("name"), "Name is required.")
    errors.required("email", form.value("email"), "Email is required.")
    errors.email("email", form.value("email"), "Enter an email address.")
    errors.required("password", form.value("password"), "Password is required.")
    role: Local String := normalizeRole(form.value("role"))
    if usersEmailTaken(form.value("email")) {
        errors.required("email", "", "That email is already registered.")
    }
    if errors.any() {
        return context.respond(
            adminUserFormView(
                context
                configuration()
                currentUserName(context)
                currentUserRole(context)
                form.old(["name", "email", "role"])
                errors
                422
            )
        )
    }
    created: Local String := usersCreate(
        form.value("name")
        form.value("email")
        form.value("password")
        role
    )
    context.flashSet("notice", "User created.")
    return context.respond(Web.redirect("/admin/users/{created}"))
}

adminUsersShow: Function := (context: RequestContext) -> Response {
    user: Local UserRecord? := usersFindByPublicID(pathPublicID(context))
    if user == null {
        return context.respond(Web.text("Not Found", 404))
    }
    return context.respond(
        adminUserShowView(
            context
            configuration()
            currentUserName(context)
            currentUserRole(context)
            user.name
            user.email
            user.role
            user.publicId
        )
    )
}

adminUsersUpdate: Function := (context: RequestContext) -> Response {
    if context.csrfValid() == false {
        return context.respond(Web.text("Forbidden", 403))
    }
    form: Local Form := context.form()
    publicId: Local String := form.value("public_id")
    user: Local UserRecord? := usersFindByPublicID(publicId)
    if user == null {
        return context.respond(Web.text("Not Found", 404))
    }
    usersUpdateName(publicId, form.value("name"))
    return context.respond(Web.redirect("/admin/users/{publicId}"))
}

adminUsersDelete: Function := (context: RequestContext) -> Response {
    if context.csrfValid() == false {
        return context.respond(Web.text("Forbidden", 403))
    }
    publicId: Local String := context.form().value("public_id")
    if publicId == currentPublicID(context) {
        context.flashSet("notice", "You cannot delete your own account.")
        return context.respond(Web.redirect("/admin/users"))
    }
    usersDelete(publicId)
    context.flashSet("notice", "User deleted.")
    return context.respond(Web.redirect("/admin/users"))
}
`

const appSQLiteUserSource = `require("Config/Database.ahd")
bring Identity
bring Security
bring SQLite
from SQLite bring SQLiteValue

UserRecord: Class<> := {
    structure: Attributes := (
        id: Int
        publicId: String
        name: String
        email: String
        passwordHash: String
        role: String
    )
}

usersFindByEmail: Function := (email: String) -> UserRecord? {
    rows: Local List<Pair<String, SQLiteValue>> := connection(
        
    ).query(
        "SELECT id, public_id, name, email, password_hash, role FROM users WHERE email = ? LIMIT 1"
        [SQLite.fromString(email)]
    )
    if len(rows) == 0 {
        return null
    }

    return usersFromRow(rows[0])
}

usersFindByPublicID: Function := (publicId: String) -> UserRecord? {
    rows: Local List<Pair<String, SQLiteValue>> := connection(
        
    ).query(
        "SELECT id, public_id, name, email, password_hash, role FROM users WHERE public_id = ? LIMIT 1"
        [SQLite.fromString(publicId)]
    )
    if len(rows) == 0 {
        return null
    }

    return usersFromRow(rows[0])
}

usersAll: Function := () -> List<UserRecord> {
    rows: Local List<Pair<String, SQLiteValue>> := connection(
        
    ).query(
        "SELECT id, public_id, name, email, password_hash, role FROM users ORDER BY name"
        []
    )
    users: Local List<UserRecord> := []
    for row in rows {
        users.add(usersFromRow(row))
    }
    return users
}

usersEmailTaken: Function := (email: String) -> Bool {
    found: Local UserRecord? := usersFindByEmail(email)
    return found != null
}

usersCreate: Function := (
    name: String
    email: String
    password: String
    role: String
) -> String {
    publicId: Local String := Identity.id()
    connection(
        
    ).execute(
        "INSERT INTO users (public_id, name, email, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))"
        [
            SQLite.fromString(publicId)
            SQLite.fromString(name)
            SQLite.fromString(email)
            SQLite.fromString(Security.passwordHash(password))
            SQLite.fromString(role)
        ]
    )
    return publicId
}

usersUpdateName: Function := (publicId: String, name: String) -> Nothing {
    connection(
        
    ).execute(
        "UPDATE users SET name = ?, updated_at = datetime('now') WHERE public_id = ?"
        [SQLite.fromString(name), SQLite.fromString(publicId)]
    )
}

usersDelete: Function := (publicId: String) -> Nothing {
    connection(
        
    ).execute(
        "DELETE FROM users WHERE public_id = ?"
        [SQLite.fromString(publicId)]
    )
}

usersFromRow: Function := (row: Pair<String, SQLiteValue>) -> UserRecord {
    return UserRecord(
        id: row["id"].int()
        publicId: row["public_id"].string()
        name: row["name"].string()
        email: row["email"].string()
        passwordHash: row["password_hash"].string()
        role: row["role"].string()
    )
}
`

const appMySQLUserSource = `require("Config/Database.ahd")
bring Identity
bring Security
bring MySQL
from MySQL bring MySQLValue

UserRecord: Class<> := {
    structure: Attributes := (
        id: Int
        publicId: String
        name: String
        email: String
        passwordHash: String
        role: String
    )
}

usersFindByEmail: Function := (email: String) -> UserRecord? {
    rows: Local List<Pair<String, MySQLValue>> := connection(
        
    ).query(
        "SELECT id, public_id, name, email, password_hash, role FROM users WHERE email = ? LIMIT 1"
        [MySQL.fromString(email)]
    )
    if len(rows) == 0 {
        return null
    }

    return usersFromRow(rows[0])
}

usersFindByPublicID: Function := (publicId: String) -> UserRecord? {
    rows: Local List<Pair<String, MySQLValue>> := connection(
        
    ).query(
        "SELECT id, public_id, name, email, password_hash, role FROM users WHERE public_id = ? LIMIT 1"
        [MySQL.fromString(publicId)]
    )
    if len(rows) == 0 {
        return null
    }

    return usersFromRow(rows[0])
}

usersAll: Function := () -> List<UserRecord> {
    rows: Local List<Pair<String, MySQLValue>> := connection(
        
    ).query(
        "SELECT id, public_id, name, email, password_hash, role FROM users ORDER BY name"
        []
    )
    users: Local List<UserRecord> := []
    for row in rows {
        users.add(usersFromRow(row))
    }
    return users
}

usersEmailTaken: Function := (email: String) -> Bool {
    found: Local UserRecord? := usersFindByEmail(email)
    return found != null
}

usersCreate: Function := (
    name: String
    email: String
    password: String
    role: String
) -> String {
    publicId: Local String := Identity.id()
    connection(
        
    ).execute(
        "INSERT INTO users (public_id, name, email, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, NOW(), NOW())"
        [
            MySQL.fromString(publicId)
            MySQL.fromString(name)
            MySQL.fromString(email)
            MySQL.fromString(Security.passwordHash(password))
            MySQL.fromString(role)
        ]
    )
    return publicId
}

usersUpdateName: Function := (publicId: String, name: String) -> Nothing {
    connection(
        
    ).execute(
        "UPDATE users SET name = ?, updated_at = NOW() WHERE public_id = ?"
        [MySQL.fromString(name), MySQL.fromString(publicId)]
    )
}

usersDelete: Function := (publicId: String) -> Nothing {
    connection(
        
    ).execute(
        "DELETE FROM users WHERE public_id = ?"
        [MySQL.fromString(publicId)]
    )
}

usersFromRow: Function := (row: Pair<String, MySQLValue>) -> UserRecord {
    return UserRecord(
        id: row["id"].int()
        publicId: row["public_id"].string()
        name: row["name"].string()
        email: row["email"].string()
        passwordHash: row["password_hash"].string()
        role: row["role"].string()
    )
}
`

const appSQLiteSchemaSQL = `CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    public_id TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
`

const appMySQLSchemaSQL = `CREATE TABLE users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    public_id VARCHAR(32) NOT NULL,
    name VARCHAR(80) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(32) NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY users_public_id (public_id),
    UNIQUE KEY users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`
