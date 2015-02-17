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

type Permission struct {
	Beacons map[string]int
}

const (
	AccessAuthLevel = 0
	ModifyAuthLevel = 1
	OwnerAuthLevel = 1
)

func (this *User) CanViewUser(otherUser User) bool {
    return this.AuthLevel > otherUser.AuthLevel
}

func (this *User) CanModifyUser(otherUser User) bool {
    return this.AuthLevel > otherUser.AuthLevel
}

func (this *User) CanAccessBeacon(beaconAddress string) bool {
	level, found := this.Permissions.Beacons[beaconAddress]
	return found && level >= AccessAuthLevel
}

func (this *User) CanModifyBeacon(beaconAddress string) bool {
	level, found := this.Permissions.Beacons[beaconAddress]
	return found && level >= ModifyAuthLevel
}