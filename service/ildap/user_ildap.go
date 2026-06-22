package ildap

import (
	"fmt"
	"strconv"

	"github.com/eryajf/go-ldap-admin/config"
	"github.com/eryajf/go-ldap-admin/model"
	"github.com/eryajf/go-ldap-admin/public/common"
	"github.com/eryajf/go-ldap-admin/public/tools"

	ldap "github.com/go-ldap/ldap/v3"
)

type UserService struct{}

// 创建资源
func (x UserService) Add(user *model.User) error {
	add := ldap.NewAddRequest(user.UserDN, nil)
	add.Attribute("objectClass", []string{"inetOrgPerson", "extensibleObject", "posixAccount"})
	add.Attribute("cn", []string{user.Nickname})
	add.Attribute("sn", []string{user.Nickname})
	add.Attribute("businessCategory", []string{user.Departments})
	add.Attribute("departmentNumber", []string{user.Position})
	add.Attribute("description", []string{user.Introduction})
	add.Attribute("displayName", []string{user.Nickname})
	add.Attribute("mail", []string{user.Mail})
	add.Attribute("employeeNumber", []string{user.JobNumber})
	add.Attribute("givenName", []string{user.GivenName})
	add.Attribute("postalAddress", []string{user.PostalAddress})
	add.Attribute("mobile", []string{user.Mobile})
	add.Attribute("uid", []string{user.Username})
	add.Attribute("employeeType", []string{user.SourceUserId})
	var pass string
	if config.Conf.Ldap.UserPasswordEncryptionType == "clear" {
		pass = tools.NewParPasswd(user.Password)
	} else {
		pass = tools.EncodePass([]byte(tools.NewParPasswd(user.Password)))
	}
	add.Attribute("userPassword", []string{pass})
	add.Attribute("uidNumber", []string{strconv.Itoa(user.UidNumber)})
	add.Attribute("gidNumber", []string{strconv.Itoa(user.GidNumber)})
	add.Attribute("homeDirectory", []string{"/home/" + user.Username})

	// 获取 LDAP 连接
	conn, err := common.GetLDAPConn()
	defer common.PutLADPConn(conn)
	if err != nil {
		return err
	}

	return conn.Add(add)
}

// Update 更新资源
func (x UserService) Update(oldusername string, user *model.User) error {
	modify := ldap.NewModifyRequest(user.UserDN, nil)
	modify.Replace("cn", []string{user.Nickname})
	modify.Replace("sn", []string{oldusername})
	modify.Replace("businessCategory", []string{user.Departments})
	modify.Replace("departmentNumber", []string{user.Position})
	modify.Replace("description", []string{user.Introduction})
	modify.Replace("displayName", []string{user.Nickname})
	modify.Replace("mail", []string{user.Mail})
	modify.Replace("employeeNumber", []string{user.JobNumber})
	modify.Replace("givenName", []string{user.GivenName})
	modify.Replace("postalAddress", []string{user.PostalAddress})
	modify.Replace("mobile", []string{user.Mobile})
	modify.Replace("employeeType", []string{user.SourceUserId})

	// 获取 LDAP 连接
	conn, err := common.GetLDAPConn()
	defer common.PutLADPConn(conn)
	if err != nil {
		return err
	}

	err = conn.Modify(modify)
	if err != nil {
		return err
	}
	if config.Conf.Ldap.UserNameModify && oldusername != user.Username {
		modifyDn := ldap.NewModifyDNRequest(fmt.Sprintf("uid=%s,%s", oldusername, config.Conf.Ldap.UserDN), fmt.Sprintf("uid=%s", user.Username), true, "")
		return conn.ModifyDN(modifyDn)
	}
	return nil
}

func (x UserService) Exist(filter map[string]any) (bool, error) {
	filter_str := ""
	for key, value := range filter {
		filter_str += fmt.Sprintf("(%s=%s)", key, value)
	}
	search_filter := fmt.Sprintf("(&(|(objectClass=inetOrgPerson)(objectClass=simpleSecurityObject))%s)", filter_str)
	// Construct query request
	searchRequest := ldap.NewSearchRequest(
		config.Conf.Ldap.BaseDN,                                     // This is basedn, we will start searching from this node.
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false, // Here several parameters are respectively scope, derefAliases, sizeLimit, timeLimit,  typesOnly
		search_filter,  // This is Filter for LDAP query
		[]string{"DN"}, // Here are the attributes returned by the query, provided as an array. If empty, all attributes are returned
		nil,
	)

	// 获取 LDAP 连接
	conn, err := common.GetLDAPConn()
	defer common.PutLADPConn(conn)
	if err != nil {
		return false, err
	}
	var sr *ldap.SearchResult
	// Search through ldap built-in search
	sr, err = conn.Search(searchRequest)
	if err != nil {
		return false, err
	}
	if len(sr.Entries) > 0 {
		return true, nil
	}
	return false, nil
}

// Delete 删除资源
func (x UserService) Delete(udn string) error {
	del := ldap.NewDelRequest(udn, nil)
	// 获取 LDAP 连接
	conn, err := common.GetLDAPConn()
	defer common.PutLADPConn(conn)
	if err != nil {
		return err
	}
	return conn.Del(del)
}

// ChangePwd 修改用户密码，此处旧密码也可以为空，ldap可以直接通过用户DN加上新密码来进行修改
func (x UserService) ChangePwd(udn, oldpasswd, newpasswd string) error {
	if config.Conf.Ldap.UserPasswordEncryptionType == "clear" {
		return updatePasswordClear(udn, newpasswd)
	}
	modifyPass := ldap.NewPasswordModifyRequest(udn, oldpasswd, newpasswd)

	// 获取 LDAP 连接
	conn, err := common.GetLDAPConn()
	defer common.PutLADPConn(conn)
	if err != nil {
		return err
	}

	_, err = conn.PasswordModify(modifyPass)
	if err != nil {
		return fmt.Errorf("password modify failed for %s, err: %v", udn, err)
	}
	return nil
}

// NewPwd 新旧密码都是空，通过管理员可以修改成功并返回新的密码
func (x UserService) NewPwd(username string) (string, error) {
	udn := fmt.Sprintf("uid=%s,%s", username, config.Conf.Ldap.UserDN)
	if username == "admin" {
		udn = config.Conf.Ldap.AdminDN
	}
	if config.Conf.Ldap.UserPasswordEncryptionType == "clear" {
		newpass := tools.GenerateRandomPassword()
		if err := updatePasswordClear(udn, newpass); err != nil {
			return "", fmt.Errorf("password modify failed for %s, err: %v", username, err)
		}
		return newpass, nil
	}
	modifyPass := ldap.NewPasswordModifyRequest(udn, "", "")

	// 获取 LDAP 连接
	conn, err := common.GetLDAPConn()
	defer common.PutLADPConn(conn)
	if err != nil {
		return "", err
	}

	newpass, err := conn.PasswordModify(modifyPass)
	if err != nil {
		return "", fmt.Errorf("password modify failed for %s, err: %v", username, err)
	}
	return newpass.GeneratedPassword, nil
}

func updatePasswordClear(udn, newpasswd string) error {
	modify := ldap.NewModifyRequest(udn, nil)
	modify.Replace("userPassword", []string{newpasswd})

	// 获取 LDAP 连接
	conn, err := common.GetLDAPConn()
	defer common.PutLADPConn(conn)
	if err != nil {
		return err
	}

	if err := conn.Modify(modify); err != nil {
		return fmt.Errorf("password modify failed for %s, err: %v", udn, err)
	}
	return nil
}
func (x UserService) ListUserDN() (users []*model.User, err error) {
	// Construct query request
	searchRequest := ldap.NewSearchRequest(
		config.Conf.Ldap.BaseDN,                                     // This is basedn, we will start searching from this node.
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false, // Here several parameters are respectively scope, derefAliases, sizeLimit, timeLimit,  typesOnly
		"(|(objectClass=inetOrgPerson)(objectClass=simpleSecurityObject))", // This is Filter for LDAP query
		[]string{"DN"}, // Here are the attributes returned by the query, provided as an array. If empty, all attributes are returned
		nil,
	)

	// 获取 LDAP 连接
	conn, err := common.GetLDAPConn()
	defer common.PutLADPConn(conn)
	if err != nil {
		return users, err
	}
	var sr *ldap.SearchResult
	// Search through ldap built-in search
	sr, err = conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}
	if len(sr.Entries) > 0 {
		for _, v := range sr.Entries {
			users = append(users, &model.User{
				UserDN: v.DN,
			})
		}
	}
	return
}

// ScanPosixIDs 扫描 LDAP 中所有用户，返回已占用的 uidNumber/gidNumber 集合，
// 以及已拥有 posixAccount 的用户 source_user_id 集合（key 为 employeeType 字段值）。
func (x UserService) ScanPosixIDs() (takenUID, takenGID map[int]bool, hasPosix map[string]bool, err error) {
	takenUID = make(map[int]bool)
	takenGID = make(map[int]bool)
	hasPosix = make(map[string]bool)

	searchRequest := ldap.NewSearchRequest(
		config.Conf.Ldap.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(|(objectClass=inetOrgPerson)(objectClass=simpleSecurityObject))",
		[]string{"objectClass", "uidNumber", "gidNumber", "employeeType"},
		nil,
	)
	conn, err := common.GetLDAPConn()
	defer common.PutLADPConn(conn)
	if err != nil {
		return
	}
	sr, err := conn.Search(searchRequest)
	if err != nil {
		return
	}
	for _, entry := range sr.Entries {
		isPosix := false
		for _, cls := range entry.GetAttributeValues("objectClass") {
			if cls == "posixAccount" {
				isPosix = true
				break
			}
		}
		if !isPosix {
			continue
		}
		if sourceUserID := entry.GetAttributeValue("employeeType"); sourceUserID != "" {
			hasPosix[sourceUserID] = true
		}
		if n, e := strconv.Atoi(entry.GetAttributeValue("uidNumber")); e == nil && n > 0 {
			takenUID[n] = true
		}
		if n, e := strconv.Atoi(entry.GetAttributeValue("gidNumber")); e == nil && n > 0 {
			takenGID[n] = true
		}
	}
	return
}

// BackfillPosixAttrs 为已存在但缺少 posixAccount 的 LDAP 条目补写 POSIX 属性。
func (x UserService) BackfillPosixAttrs(udn string, uidNum, gidNum int, username string) error {
	modify := ldap.NewModifyRequest(udn, nil)
	modify.Add("objectClass", []string{"posixAccount"})
	modify.Add("uidNumber", []string{strconv.Itoa(uidNum)})
	modify.Add("gidNumber", []string{strconv.Itoa(gidNum)})
	modify.Add("homeDirectory", []string{"/home/" + username})

	conn, err := common.GetLDAPConn()
	defer common.PutLADPConn(conn)
	if err != nil {
		return err
	}
	return conn.Modify(modify)
}
