// Copyright (c) 2017 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package project

import (
	"fmt"

	"github.com/vmware/harbor/src/common/dao"
	"github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/common/utils/log"
)

// GetProjectMember gets all members of the project.
func GetProjectMember(queryMember models.Member) ([]*models.Member, error) {
	log.Debugf("Query condition %+v", queryMember)
	if queryMember.ProjectID == 0 {
		return nil, fmt.Errorf("Failed to query project member, query condition %v", queryMember)
	}

	o := dao.GetOrmer()
	sql := ` select a.* from ((select pm.id as id, pm.project_id as project_id, ug.id as entity_id, ug.group_name as entity_name, ug.creation_time, ug.update_time, r.name as rolename, 
		r.role_id as role, pm.entity_type as entity_type from user_group ug join project_member pm 
		on pm.project_id = ? and ug.id = pm.entity_id join role r on pm.role = r.role_id where  pm.entity_type = 'g')
		union
		(select pm.id as id, pm.project_id as project_id, u.user_id as entity_id, u.username as entity_name, u.creation_time, u.update_time, r.name as rolename, 
		r.role_id as role, pm.entity_type as entity_type from user u join project_member pm 
		on pm.project_id = ? and u.user_id = pm.entity_id 
		join role r on pm.role = r.role_id where u.deleted = 0 and pm.entity_type = 'u')) as a where a.project_id = ? `

	queryParam := make([]interface{}, 1)
	// used ProjectID already
	queryParam = append(queryParam, queryMember.ProjectID)
	queryParam = append(queryParam, queryMember.ProjectID)
	queryParam = append(queryParam, queryMember.ProjectID)

	if len(queryMember.Entityname) > 0 {
		sql += " and a.entity_name = ? "
		queryParam = append(queryParam, queryMember.Entityname)
	}

	if len(queryMember.EntityType) == 1 {
		sql += " and a.entity_type = ? "
		queryParam = append(queryParam, queryMember.EntityType)
	}

	if queryMember.EntityID > 0 {
		sql += " and a.entity_id = ? "
		queryParam = append(queryParam, queryMember.EntityID)
	}
	if queryMember.ID > 0 {
		sql += " and a.id = ? "
		queryParam = append(queryParam, queryMember.ID)
	}
	sql += ` order by a.entity_name `
	members := []*models.Member{}
	_, err := o.Raw(sql, queryParam).QueryRows(&members)

	return members, err
}

// AddProjectMember inserts a record to table project_member
func AddProjectMember(member models.Member) (int, error) {

	log.Debugf("Adding project member %+v", member)
	o := dao.GetOrmer()

	if member.EntityID <= 0 {
		return 0, fmt.Errorf("Invalid entity_id, member: %+v", member)
	}

	if member.ProjectID <= 0 {
		return 0, fmt.Errorf("Invalid project_id, member: %+v", member)
	}

	delSQL := "delete from project_member where project_id = ? and entity_id = ? and entity_type = ? "
	_, err := o.Raw(delSQL, member.ProjectID, member.EntityID, member.EntityType).Exec()
	if err != nil {
		return 0, err
	}
	sql := "insert into project_member (project_id, entity_id , role, entity_type) values (?, ?, ?, ?)"
	r, err := o.Raw(sql, member.ProjectID, member.EntityID, member.Role, member.EntityType).Exec()
	if err != nil {
		return 0, err
	}
	pmid, err := r.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(pmid), err
}

// UpdateProjectMemberRole updates the record in table project_member, only role can be changed
func UpdateProjectMemberRole(pmID int, role int) error {
	o := dao.GetOrmer()
	sql := "update project_member set role = ? where id = ? "
	_, err := o.Raw(sql, role, pmID).Exec()
	return err
}

// DeleteProjectMemberByID - Delete Project Member by ID
func DeleteProjectMemberByID(pmid int) error {
	o := dao.GetOrmer()
	sql := "delete from project_member where id = ?"
	if _, err := o.Raw(sql, pmid).Exec(); err != nil {
		return err
	}
	return nil
}

// SearchMemberByName search members of the project by entity_name
func SearchMemberByName(projectID int64, entityName string) ([]*models.Member, error) {
	o := dao.GetOrmer()
	sql := `(select pm.id, pm.project_id, 
	               u.username as entity_name, 
	               r.name as rolename,
			       pm.role, pm.entity_id, pm.entity_type 
			  from project_member pm
         left join user u on pm.entity_id = u.user_id and pm.entity_type = 'u'
		 left join role r on pm.role = r.role_id
			 where u.deleted = 0 and pm.project_id = ? and u.username like ? order by entity_name )
			union
		   (select pm.id, pm.project_id, 
			       ug.group_name as entity_name, 
				   r.name as rolename,
				   pm.role, pm.entity_id, pm.entity_type 
		      from project_member pm
	     left join user_group ug on pm.entity_id = ug.id and pm.entity_type = 'g'
	     left join role r on pm.role = r.role_id
		     where pm.project_id = ? and ug.group_name like ? order by entity_name ) `
	queryParam := make([]interface{}, 4)
	queryParam = append(queryParam, projectID)
	queryParam = append(queryParam, "%"+dao.Escape(entityName)+"%")
	queryParam = append(queryParam, projectID)
	queryParam = append(queryParam, "%"+dao.Escape(entityName)+"%")
	members := []*models.Member{}
	_, err := o.Raw(sql, queryParam).QueryRows(&members)
	return members, err
}
