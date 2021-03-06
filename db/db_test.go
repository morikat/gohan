// Copyright (C) 2015 NTT Innovation Institute, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package db_test

import (
	"os"

	"github.com/cloudwan/gohan/db"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Database operation test", func() {
	var (
		err error
		ok  bool

		conn   string
		dbType string

		manager          *schema.Manager
		networkSchema    *schema.Schema
		serverSchema     *schema.Schema
		networkResource1 *schema.Resource
		networkResource2 *schema.Resource
		subnetResource   *schema.Resource
		serverResource   *schema.Resource

		dataStore db.DB
	)

	BeforeEach(func() {
		manager = schema.GetManager()
		Expect(manager.LoadSchemaFromFile("../etc/schema/gohan.json")).To(Succeed())
	})

	AfterEach(func() {
		schema.ClearManager()
		if os.Getenv("MYSQL_TEST") != "true" {
			os.Remove(conn)
		}
	})

	Describe("Base operations", func() {
		var (
			tx transaction.Transaction
		)

		BeforeEach(func() {
			Expect(manager.LoadSchemaFromFile("../tests/test_abstract_schema.yaml")).To(Succeed())
			Expect(manager.LoadSchemaFromFile("../tests/test_schema.yaml")).To(Succeed())
			networkSchema, ok = manager.Schema("network")
			Expect(ok).To(BeTrue())
			_, ok = manager.Schema("subnet")
			Expect(ok).To(BeTrue())
			serverSchema, ok = manager.Schema("server")
			Expect(ok).To(BeTrue())

			network1 := map[string]interface{}{
				"id":                "networkRed",
				"name":              "NetworkRed",
				"description":       "A crimson network",
				"tenant_id":         "red",
				"shared":            false,
				"route_targets":     []string{"1000:10000", "2000:20000"},
				"providor_networks": map[string]interface{}{"segmentation_id": 10, "segmentation_type": "vlan"}}
			networkResource1, err = manager.LoadResource("network", network1)
			Expect(err).ToNot(HaveOccurred())

			network2 := map[string]interface{}{
				"id":                "networkBlue",
				"name":              "NetworkBlue",
				"description":       "A crimson network",
				"tenant_id":         "blue",
				"shared":            false,
				"route_targets":     []string{"1000:10000", "2000:20000"},
				"providor_networks": map[string]interface{}{"segmentation_id": 10, "segmentation_type": "vlan"}}
			networkResource2, err = manager.LoadResource("network", network2)
			Expect(err).ToNot(HaveOccurred())

			subnet := map[string]interface{}{
				"id":          "subnetRed",
				"name":        "SubnetRed",
				"description": "A crimson subnet",
				"tenant_id":   "red",
				"cidr":        "10.0.0.0/24"}
			subnetResource, err = manager.LoadResource("subnet", subnet)
			Expect(err).ToNot(HaveOccurred())
			subnetResource.SetParentID("networkRed")
			Expect(subnetResource.Path()).To(Equal("/v2.0/subnets/subnetRed"))

			server := map[string]interface{}{
				"id":          "serverRed",
				"name":        "serverRed",
				"tenant_id":   "red",
				"network_id":  "networkRed",
				"description": "red server",
				"cidr":        "10.0.0.0/24"}
			serverResource, err = manager.LoadResource("server", server)
			Expect(err).ToNot(HaveOccurred())
		})

		JustBeforeEach(func() {
			os.Remove(conn)
			dataStore, err = db.ConnectDB(dbType, conn, db.DefaultMaxOpenConn)
			Expect(err).ToNot(HaveOccurred())

			for _, s := range manager.Schemas() {
				Expect(dataStore.RegisterTable(s, false)).To(Succeed())
			}

			tx, err = dataStore.Begin()
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			tx.Close()
		})

		Describe("Using sql", func() {
			BeforeEach(func() {
				if os.Getenv("MYSQL_TEST") == "true" {
					conn = "root@/gohan_test"
					dbType = "mysql"
				} else {
					conn = "./test.db"
					dbType = "sqlite3"
				}
			})

			Context("When the database is empty", func() {
				It("Returns an empty list", func() {
					list, num, err := tx.List(networkSchema, nil, nil)
					Expect(err).ToNot(HaveOccurred())
					Expect(num).To(Equal(uint64(0)))
					Expect(list).To(BeEmpty())
					Expect(tx.Commit()).To(Succeed())
				})

				It("Creates a resource", func() {
					Expect(tx.Create(networkResource1)).To(Succeed())

					Expect(tx.Commit()).To(Succeed())
				})
			})

			Describe("When the database is not empty", func() {
				JustBeforeEach(func() {
					Expect(tx.Create(networkResource1)).To(Succeed())
					Expect(tx.Create(networkResource2)).To(Succeed())
					Expect(tx.Create(serverResource)).To(Succeed())
					Expect(tx.Commit()).To(Succeed())
					tx.Close()
					tx, err = dataStore.Begin()
					Expect(err).ToNot(HaveOccurred())
				})

				It("Returns the expected list", func() {
					list, num, err := tx.List(networkSchema, nil, nil)
					Expect(err).ToNot(HaveOccurred())
					Expect(num).To(Equal(uint64(2)))
					Expect(list).To(HaveLen(2))
					Expect(list[0]).To(util.MatchAsJSON(networkResource1))
					Expect(list[1]).To(util.MatchAsJSON(networkResource2))
					Expect(tx.Commit()).To(Succeed())
				})

				It("Returns the expected list with filter", func() {
					filter := map[string]interface{}{
						"tenant_id": []string{"red"},
					}
					list, num, err := tx.List(networkSchema, filter, nil)
					Expect(err).ToNot(HaveOccurred())
					Expect(num).To(Equal(uint64(1)))
					Expect(list).To(HaveLen(1))
					Expect(list[0]).To(util.MatchAsJSON(networkResource1))
					Expect(tx.Commit()).To(Succeed())
				})

				It("Returns the error with invalid filter", func() {
					filter := map[string]interface{}{
						"bad_filter": []string{"red"},
					}
					_, _, err := tx.List(networkSchema, filter, nil)
					Expect(err).To(HaveOccurred())
				})

				It("Shows related resources", func() {
					list, num, err := tx.List(serverSchema, nil, nil)
					Expect(err).ToNot(HaveOccurred())
					Expect(num).To(Equal(uint64(1)))
					Expect(list).To(HaveLen(1))
					Expect(list[0].Data()).To(HaveKeyWithValue("network", HaveKeyWithValue("name", networkResource1.Data()["name"])))
					Expect(tx.Commit()).To(Succeed())
				})

				It("Fetches an existing resource", func() {
					networkResourceFetched, err := tx.Fetch(networkSchema, transaction.IDFilter(networkResource1.ID()))
					Expect(err).ToNot(HaveOccurred())
					Expect(networkResourceFetched).To(util.MatchAsJSON(networkResource1))
					Expect(tx.Commit()).To(Succeed())
				})

				It("Updates the resource properly", func() {
					By("Not allowing to update some fields")
					Expect(networkResource1.Update(map[string]interface{}{"id": "new_id"})).ToNot(Succeed())

					By("Updating other fields")
					Expect(networkResource1.Update(map[string]interface{}{"name": "new_name"})).To(Succeed())
					Expect(tx.Update(networkResource1)).To(Succeed())
					Expect(tx.Commit()).To(Succeed())
				})

				It("Creates a dependent resource", func() {
					Expect(tx.Create(subnetResource)).To(Succeed())
					Expect(tx.Commit()).To(Succeed())
				})

				It("Deletes the resource", func() {
					Expect(tx.Delete(serverSchema, serverResource.ID())).To(Succeed())
					Expect(tx.Delete(networkSchema, networkResource1.ID())).To(Succeed())
					Expect(tx.Commit()).To(Succeed())
				})

				Context("Using StateFetch", func() {
					It("Returns the defaults", func() {
						beforeState, err := tx.StateFetch(networkSchema, transaction.IDFilter(networkResource1.ID()))
						Expect(err).ToNot(HaveOccurred())
						Expect(tx.Commit()).To(Succeed())
						Expect(beforeState.ConfigVersion).To(Equal(int64(1)))
						Expect(beforeState.StateVersion).To(Equal(int64(0)))
						Expect(beforeState.State).To(Equal(""))
						Expect(beforeState.Error).To(Equal(""))
						Expect(beforeState.Monitoring).To(Equal(""))
					})
				})
			})
		})
	})

	Context("Initialization", func() {
		BeforeEach(func() {
			conn = "test.db"
			dbType = "sqlite3"
			Expect(manager.LoadSchemaFromFile("../tests/test_abstract_schema.yaml")).To(Succeed())
			Expect(manager.LoadSchemaFromFile("../tests/test_schema.yaml")).To(Succeed())
		})

		It("Should initialize the database without error", func() {
			Expect(db.InitDBWithSchemas(dbType, conn, false, false)).To(Succeed())
		})
	})

	Context("Converting", func() {
		BeforeEach(func() {
			Expect(manager.LoadSchemaFromFile("test_data/conv_in.yaml")).To(Succeed())
		})

		It("Should do it properly", func() {
			inDB, err := db.ConnectDB("yaml", "test_data/conv_in.yaml", db.DefaultMaxOpenConn)
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove("test_data/conv_in.db")

			db.InitDBWithSchemas("sqlite3", "test_data/conv_out.db", false, false)
			outDB, err := db.ConnectDB("sqlite3", "test_data/conv_out.db", db.DefaultMaxOpenConn)
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove("test_data/conv_out.db")

			db.InitDBWithSchemas("yaml", "test_data/conv_verify.yaml", false, false)
			verifyDB, err := db.ConnectDB("yaml", "test_data/conv_verify.yaml", db.DefaultMaxOpenConn)
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove("test_data/conv_verify.yaml")

			Expect(db.CopyDBResources(inDB, outDB, true)).To(Succeed())

			Expect(db.CopyDBResources(outDB, verifyDB, true)).To(Succeed())

			inTx, err := inDB.Begin()
			Expect(err).ToNot(HaveOccurred())
			defer inTx.Close()

			// SQL returns different types than JSON/YAML Database
			// So we need to move it back again so that DeepEqual would work correctly
			verifyTx, err := verifyDB.Begin()
			Expect(err).ToNot(HaveOccurred())
			defer verifyTx.Close()

			for _, s := range manager.OrderedSchemas() {
				if s.Metadata["type"] == "metaschema" {
					continue
				}
				resources, _, err := inTx.List(s, nil, nil)
				Expect(err).ToNot(HaveOccurred())
				for _, inResource := range resources {
					outResource, err := verifyTx.Fetch(s, transaction.Filter{"id": inResource.ID()})
					Expect(err).ToNot(HaveOccurred())
					Expect(outResource).To(Equal(inResource))
				}
			}
		})

		It("Should not override existing rows", func() {
			inDB, err := db.ConnectDB("yaml", "test_data/conv_in.yaml", db.DefaultMaxOpenConn)
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove("test_data/conv_in.db")

			db.InitDBWithSchemas("sqlite3", "test_data/conv_out.db", false, false)
			outDB, err := db.ConnectDB("sqlite3", "test_data/conv_out.db", db.DefaultMaxOpenConn)
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove("test_data/conv_out.db")

			Expect(err).ToNot(HaveOccurred())
			defer os.Remove("test_data/conv_verify.yaml")

			Expect(db.CopyDBResources(inDB, outDB, false)).To(Succeed())
			subnetSchema, _ := manager.Schema("subnet")

			// Update some data
			tx, err := outDB.Begin()
			Expect(err).ToNot(HaveOccurred())
			list, _, err := tx.List(subnetSchema, map[string]interface{}{
				"name": "subnetRedA",
			}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			subnet := list[0]
			subnet.Data()["description"] = "Updated description"
			err = tx.Update(subnet)
			Expect(err).ToNot(HaveOccurred())
			tx.Commit()
			tx.Close()

			Expect(db.CopyDBResources(inDB, outDB, false)).To(Succeed())
			// check description of subnetRedA
			tx, err = outDB.Begin()
			Expect(err).ToNot(HaveOccurred())
			list, _, err = tx.List(subnetSchema, map[string]interface{}{
				"name": "subnetRedA",
			}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(list).To(HaveLen(1))
			subnet = list[0]
			Expect(subnet.Data()["description"]).To(Equal("Updated description"))
			tx.Close()
		})
	})
})
