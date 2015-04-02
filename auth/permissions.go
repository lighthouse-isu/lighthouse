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
		"Beacons" : make(map[interface{}]interface{}),
		"Applications" : make(map[interface{}]interface{})
	}
}

func (this *User) convertPermissionsFromDB() {
	for _, permInter := range this.Permissions {

		permSet := permInter.(map[interface{}]interface{})

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

func (this *User) GetAuthLevel(field string, key interface{}) int {
	permMap, ok := this.Permissions[field]
	if !ok {
		return -1
	}

	val, ok := permMap.(map[interface{}]interface{})[key]
	if !ok {
		return -1
	}

	return val.(int)
}

func (this *User) SetAuthLevel(field string, key interface{}, level int) {
	fieldInter, ok := this.Permissions[field]
	if !ok {
		return
	}

	fieldVal, _ := fieldInter.(map[interface{}]interface{})
	if fieldVal == nil {
		fieldVal = make(map[interface{}]interface{})
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

func (this *User) CanAccessApplication(app int64) bool {
	level := this.GetAuthLevel("Applications", app)
	return level >= AccessAuthLevel
}

func (this *User) CanModifyApplcation(app int64) bool {
	level := this.GetAuthLevel("Applications", app)
	return level >= ModifyAuthLevel
}