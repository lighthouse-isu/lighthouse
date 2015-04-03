// Copyright 2014 Caleb Brose, Chris Fogerty, Rob Sheehy, Zach Taylor, Nick Miller
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

type Permission map[string]interface{}

const (
	AccessAuthLevel = 0
	ModifyAuthLevel = 1
	OwnerAuthLevel  = 2
)

func NewPermission() Permission {
	return Permission{
		"Beacons" : make(map[string]interface{}),
		"Applications" : make(map[string]interface{}),
	}
}

func (this *User) convertPermissionsFromDB() {
	for _, permInter := range this.Permissions {

		permSet := permInter.(map[string]interface{})

		for perm, level := range permSet {
			val, ok := level.(float64)
			if ok {
				permSet[perm] = int(val)
			} else {
				permSet[perm] = level.(int)
			}
		} 
	}
}

func (this *User) GetAuthLevel(field, key string) int {
	permMap, ok := this.Permissions[field]
	if !ok {
		return -1
	}

	val, ok := permMap.(map[string]interface{})[key]
	if !ok {
		return -1
	}

	return val.(int)
}

func (this *User) SetAuthLevel(field, key string, level int) {
	fieldInter, ok := this.Permissions[field]
	if !ok {
		return
	}

	fieldVal, _ := fieldInter.(map[string]interface{})
	if fieldVal == nil {
		fieldVal = make(map[string]interface{})
	}

	if level < DefaultAuthLevel {
		delete(fieldVal, key)
	} else {
		fieldVal[key] = level
	}

	this.Permissions[field] = fieldVal
}

func (this *User) CanViewUser(otherUser *User) bool {
    return this.Email == otherUser.Email || 
    	this.AuthLevel > otherUser.AuthLevel
}

func (this *User) CanModifyUser(otherUser *User) bool {
    return this.Email == otherUser.Email || 
    	this.AuthLevel > otherUser.AuthLevel
}

func (this *User) CanAccessBeacon(beaconAddress string) bool {
	level := this.GetAuthLevel("Beacons", beaconAddress)
	return level >= AccessAuthLevel
}

func (this *User) CanModifyBeacon(beaconAddress string) bool {
	level := this.GetAuthLevel("Beacons", beaconAddress)
	return level >= ModifyAuthLevel
}

func SetUserBeaconAuthLevel(user *User, beacon string, level int) error {
    user.SetAuthLevel("Beacons", beacon, level)
    
    to := map[string]interface{}{"Permissions" : user.Permissions}
    where := map[string]interface{}{"Email" : user.Email}

    return users.Update(to, where)
}

func (this *User) CanAccessApplication(name string) bool {
	level := this.GetAuthLevel("Applications", name)
	return level >= AccessAuthLevel
}

func (this *User) CanModifyApplcation(name string) bool {
	level := this.GetAuthLevel("Applications", name)
	return level >= ModifyAuthLevel
}

func SetUserApplicationAuthLevel(user *User, name string, level int) error {
    user.SetAuthLevel("Applications", name, level)
    
    to := map[string]interface{}{"Permissions" : user.Permissions}
    where := map[string]interface{}{"Email" : user.Email}

    return users.Update(to, where)
}