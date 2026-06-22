package ildap

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/eryajf/go-ldap-admin/config"
	"github.com/eryajf/go-ldap-admin/model"
	"github.com/eryajf/go-ldap-admin/public/common"
	"github.com/eryajf/go-ldap-admin/public/tools"

	ldap "github.com/go-ldap/ldap/v3"
)

type GroupService struct{}

// Add 添加资源
func (x GroupService) Add(g *model.Group) error { //organizationalUnit
	if g.Remark == "" {
		g.Remark = g.GroupName
	}
	add := ldap.NewAddRequest(g.GroupDN, nil)
	if g.GroupType == "ou" {
		add.Attribute("objectClass", []string{"organizationalUnit", "top"})
	}
	if g.GroupType == "cn" {
		add.Attribute("objectClass", []string{"groupOfUniqueNames", "top"})
		add.Attribute("uniqueMember", []string{config.Conf.Ldap.AdminDN})
	}
	add.Attribute(g.GroupType, []string{g.GroupName})
	add.Attribute("description", []string{g.Remark})

	// 获取 LDAP 连接
	conn, err := common.GetLDAPConn()
	defer common.PutLADPConn(conn)
	if err != nil {
		return err
	}

	return conn.Add(add)
}

// UpdateGroup 更新一个分组
func (x GroupService) Update(oldGroup, newGroup *model.Group) error {
	modify1 := ldap.NewModifyRequest(oldGroup.GroupDN, nil)
	modify1.Replace("description", []string{newGroup.Remark})

	// 获取 LDAP 连接
	conn, err := common.GetLDAPConn()
	defer common.PutLADPConn(conn)
	if err != nil {
		return err
	}

	err = conn.Modify(modify1)
	if err != nil {
		return err
	}
	// 如果配置文件允许修改分组名称，且分组名称发生了变化，那么执行修改分组名称
	if config.Conf.Ldap.GroupNameModify && newGroup.GroupName != oldGroup.GroupName {
		modify2 := ldap.NewModifyDNRequest(oldGroup.GroupDN, newGroup.GroupDN, true, "")
		err := conn.ModifyDN(modify2)
		if err != nil {
			return err
		}
	}
	return nil
}

// Delete 删除资源
func (x GroupService) Delete(gdn string) error {
	del := ldap.NewDelRequest(gdn, nil)

	// 获取 LDAP 连接
	conn, err := common.GetLDAPConn()
	defer common.PutLADPConn(conn)
	if err != nil {
		return err
	}

	return conn.Del(del)
}

// AddUserToGroup 添加用户到分组
func (x GroupService) AddUserToGroup(dn, udn string) error {
	if dn[:3] == "ou=" {
		return tools.NewLdapI18nError("group.ou_cannot_add_user", nil)
	}
	newmr := ldap.NewModifyRequest(dn, nil)
	newmr.Add("uniqueMember", []string{udn})

	conn, err := common.GetLDAPConn()
	defer common.PutLADPConn(conn)
	if err != nil {
		return err
	}
	return conn.Modify(newmr)
}

// RemoveUserFromGroup 将用户从分组删除
func (x GroupService) RemoveUserFromGroup(gdn, udn string) error {
	newmr := ldap.NewModifyRequest(gdn, nil)
	newmr.Delete("uniqueMember", []string{udn})

	conn, err := common.GetLDAPConn()
	defer common.PutLADPConn(conn)
	if err != nil {
		return err
	}
	return conn.Modify(newmr)
}

// uidFromDN extracts the uid value from a DN like "uid=zhangw,ou=people,...".
func uidFromDN(udn string) string {
	parts := strings.SplitN(udn, ",", 2)
	kv := strings.SplitN(parts[0], "=", 2)
	if len(kv) != 2 {
		return ""
	}
	return kv[1]
}

// DelUserFromGroup 将用户从分组删除
func (x GroupService) ListGroupDN() (groups []*model.Group, err error) {
	// Construct query request
	searchRequest := ldap.NewSearchRequest(
		config.Conf.Ldap.BaseDN,                                     // This is basedn, we will start searching from this node.
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false, // Here several parameters are respectively scope, derefAliases, sizeLimit, timeLimit,  typesOnly
		"(|(objectClass=organizationalUnit)(objectClass=groupOfUniqueNames))", // This is Filter for LDAP query
		[]string{"DN"}, // Here are the attributes returned by the query, provided as an array. If empty, all attributes are returned
		nil,
	)

	// 获取 LDAP 连接
	conn, err := common.GetLDAPConn()
	defer common.PutLADPConn(conn)
	if err != nil {
		return groups, err
	}
	var sr *ldap.SearchResult
	// Search through ldap built-in search
	sr, err = conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}
	if len(sr.Entries) > 0 {
		for _, v := range sr.Entries {
			groups = append(groups, &model.Group{
				GroupDN: v.DN,
			})
		}
	}
	return
}

// nasGroupDN 返回 NAS 专用组的 DN。
func nasGroupDN(groupName string) string {
	return fmt.Sprintf("cn=%s,ou=nas-groups,%s", groupName, config.Conf.Ldap.BaseDN)
}

// EnsureNasGroupsOU 确保 ou=nas-groups 容器存在，不存在则创建。
func (x GroupService) EnsureNasGroupsOU() error {
	ouDN := fmt.Sprintf("ou=nas-groups,%s", config.Conf.Ldap.BaseDN)
	conn, err := common.GetLDAPConn()
	defer common.PutLADPConn(conn)
	if err != nil {
		return err
	}

	sr, err := conn.Search(ldap.NewSearchRequest(
		ouDN, ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=*)", []string{"dn"}, nil,
	))
	if err == nil && len(sr.Entries) > 0 {
		return nil
	}

	add := ldap.NewAddRequest(ouDN, nil)
	add.Attribute("objectClass", []string{"organizationalUnit", "top"})
	add.Attribute("ou", []string{"nas-groups"})
	return conn.Add(add)
}

// SyncNasGroup 在 ou=nas-groups 下为部门创建或更新对应的 posixGroup 条目。
// memberUid 从现有 groupOfUniqueNames 条目的 uniqueMember 中提取。
func (x GroupService) SyncNasGroup(g *model.Group) error {
	conn, err := common.GetLDAPConn()
	defer common.PutLADPConn(conn)
	if err != nil {
		return err
	}

	// 从已有的 groupOfUniqueNames 条目读取 uniqueMember，提取 uid 列表
	var memberUids []string
	if sr, e := conn.Search(ldap.NewSearchRequest(
		g.GroupDN, ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=*)", []string{"uniqueMember"}, nil,
	)); e == nil && len(sr.Entries) > 0 {
		for _, member := range sr.Entries[0].GetAttributeValues("uniqueMember") {
			if uid := uidFromDN(member); uid != "" {
				// 过滤掉 cn=admin 之类的非 uid= 占位条目
				if strings.HasPrefix(strings.ToLower(member), "uid=") {
					memberUids = append(memberUids, uid)
				}
			}
		}
	}

	nasDN := nasGroupDN(g.GroupName)
	gidStr := strconv.Itoa(g.GidNumber)

	// 检查 NAS 组是否已存在
	existSr, err := conn.Search(ldap.NewSearchRequest(
		nasDN, ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=*)", []string{"cn"}, nil,
	))

	if err != nil || len(existSr.Entries) == 0 {
		// 不存在则新建
		add := ldap.NewAddRequest(nasDN, nil)
		add.Attribute("objectClass", []string{"posixGroup", "top"})
		add.Attribute("cn", []string{g.GroupName})
		add.Attribute("gidNumber", []string{gidStr})
		if len(memberUids) > 0 {
			add.Attribute("memberUid", memberUids)
		}
		return conn.Add(add)
	}

	// 已存在则更新 gidNumber 和 memberUid
	modify := ldap.NewModifyRequest(nasDN, nil)
	modify.Replace("gidNumber", []string{gidStr})
	if len(memberUids) > 0 {
		modify.Replace("memberUid", memberUids)
	}
	return conn.Modify(modify)
}
