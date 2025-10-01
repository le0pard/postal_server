package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	gopostalExpand "github.com/openvenues/gopostal/expand"
	gopostalParser "github.com/openvenues/gopostal/parser"

	ginzerolog "github.com/dn365/gin-zerolog"
	"github.com/gin-gonic/gin"
	"github.com/le0pard/postal_server/version"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// EnvPrefix for environment variables
const EnvPrefix string = "POSTAL_SERVER"

var (
	cfgFile                      string
	EnvStrReplacer               = strings.NewReplacer(".", "_")
	Version                      = fmt.Sprintf("%s, date %s, build %s", version.Version, version.BuildTime, version.GitCommit)
	queryParamToAddressComponent = map[string]uint16{
		"address_name":         gopostalExpand.AddressName,
		"address_house_number": gopostalExpand.AddressHouseNumber,
		"address_street":       gopostalExpand.AddressStreet,
		"address_po_box":       gopostalExpand.AddressPoBox,
		"address_unit":         gopostalExpand.AddressUnit,
		"address_level":        gopostalExpand.AddressLevel,
		"address_entrance":     gopostalExpand.AddressEntrance,
		"address_staircase":    gopostalExpand.AddressStaircase,
		"address_postal_code":  gopostalExpand.AddressPostalCode,
	}
)

func mapQueryParamsOnExpandOptions(options gopostalExpand.ExpandOptions, queryParams url.Values) gopostalExpand.ExpandOptions {
	for key, values := range queryParams {
		switch strings.ToLower(key) {
		case "languages":
			options.Languages = values
		case "latin_ascii": // true
			options.LatinAscii = stringToBool(values[0])
		case "transliterate": // true
			options.Transliterate = stringToBool(values[0])
		case "strip_accents": // true
			options.StripAccents = stringToBool(values[0])
		case "lowercase": // true
			options.Lowercase = stringToBool(values[0])
		case "trim_string": // true
			options.TrimString = stringToBool(values[0])
		case "replace_word_hyphens": // true
			options.ReplaceWordHyphens = stringToBool(values[0])
		case "delete_word_hyphens": // true
			options.DeleteWordHyphens = stringToBool(values[0])
		case "replace_numeric_hyphens": // false
			options.ReplaceNumericHyphens = stringToBool(values[0])
		case "delete_numeric_hyphens": // false
			options.DeleteNumericHyphens = stringToBool(values[0])
		case "split_alpha_from_numeric": // true
			options.SplitAlphaFromNumeric = stringToBool(values[0])
		case "delete_final_periods": // true
			options.DeleteFinalPeriods = stringToBool(values[0])
		case "delete_acronym_periods": // true
			options.DeleteAcronymPeriods = stringToBool(values[0])
		case "drop_english_possessives": // true
			options.DropEnglishPossessives = stringToBool(values[0])
		case "delete_apostrophes": // true
			options.DeleteApostrophes = stringToBool(values[0])
		case "expand_numex": // true
			options.ExpandNumex = stringToBool(values[0])
		case "roman_numerals": // true
			options.RomanNumerals = stringToBool(values[0])
		}
	}

	if newComponents, found := parseAddressComponents(queryParams); found {
		options.AddressComponents = newComponents
	}

	return options
}

func parseAddressComponents(queryParams url.Values) (uint16, bool) {
	var components uint16 = gopostalExpand.AddressNone
	var found bool = false

	for key := range queryParams {
		// Look up the component constant for the given query parameter key.
		if component, ok := queryParamToAddressComponent[key]; ok {
			found = true
			// If found, combine it with the existing components using bitwise OR.
			components |= component
		}
	}

	return components, found
}

func stringToBool(s string) bool {
	if s != "" {
		if boolValue, err := strconv.ParseBool(s); err == nil {
			return boolValue
		}
	}
	return false
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:                   "postal_server",
	Short:                 "Postal web server offers advanced capabilities for parsing and standardizing street addresses",
	Version:               Version,
	SilenceUsage:          true,
	SilenceErrors:         true,
	DisableFlagsInUseLine: true,
	TraverseChildren:      true,
	Long:                  `Postal web server that grants access to the libpostal library, enabling the parsing and normalization of street addresses globally`,
	Run: func(cmd *cobra.Command, args []string) {
		r := gin.New()

		r.UseH2C = viper.GetBool("h2c")
		r.Use(gin.Recovery())
		r.Use(ginzerolog.Logger("postal_server"))
		if viper.IsSet("trusted_proxies") {
			r.SetTrustedProxies(viper.GetStringSlice("trusted_proxies"))
		}

		// healthcheck endpoint
		r.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status": "ok",
			})
		})

		// basic auth
		if viper.IsSet("basic_auth_username") && viper.IsSet("basic_auth_password") {
			r.Use(gin.BasicAuth(gin.Accounts{
				viper.GetString("basic_auth_username"): viper.GetString("basic_auth_password"),
			}))
		}
		// bearer token auth
		if viper.IsSet("bearer_auth_token") {
			r.Use(MiddlewareWithStaticToken(viper.GetString("bearer_auth_token")))
		}

		// expand libpostal
		r.GET("/expand", func(c *gin.Context) {
			queryParams := c.Request.URL.Query()
			address := c.DefaultQuery("address", "")

			options := gopostalExpand.GetDefaultExpansionOptions()
			expansions := gopostalExpand.ExpandAddressOptions(
				address,
				mapQueryParamsOnExpandOptions(
					options,
					queryParams,
				),
			)
			c.JSON(http.StatusOK, expansions)
		})

		// parse libpostal
		r.GET("/parse", func(c *gin.Context) {
			address := c.DefaultQuery("address", "")
			language := c.DefaultQuery("language", "")
			country := c.DefaultQuery("country", "")

			parsed := gopostalParser.ParseAddressOptions(
				address,
				gopostalParser.ParserOptions{
					Language: language,
					Country:  country,
				},
			)
			c.JSON(http.StatusOK, parsed)
		})

		// root
		r.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"version": Version,
			})
		})

		r.Run(fmt.Sprintf("%s:%d", viper.GetString("host"), viper.GetInt("port")))
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func initLogging() {
	if viper.IsSet("log_format") && strings.ToLower(viper.GetString("log_format")) == "json" {
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if viper.GetBool("debug") {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		return
	}

	if viper.IsSet("log_level") {
		var level, err = zerolog.ParseLevel(viper.GetString("log_level"))
		if err == nil {
			zerolog.SetGlobalLevel(level)
		} else {
			log.Warn().
				Err(err).
				Str("level", viper.GetString("log_level")).
				Msg("Invalid log level")
		}
		return
	}
}

func initConfig() {
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(EnvStrReplacer)

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".postal_server" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".postal_server")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	initLogging()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.postal_server.yaml)")

	rootCmd.SetVersionTemplate("PostalServer version {{.Version}}\n")

	rootCmd.PersistentFlags().Bool("debug", false, "use debug logging")
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))

	rootCmd.PersistentFlags().Bool("h2c", false, "whether to use http2 h2c, default false")
	viper.BindPFlag("h2c", rootCmd.PersistentFlags().Lookup("h2c"))

	rootCmd.PersistentFlags().StringP("host", "H", "0.0.0.0", "server host")
	viper.BindPFlag("host", rootCmd.PersistentFlags().Lookup("host"))
	rootCmd.PersistentFlags().IntP("port", "p", 8000, "server port")
	viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))
	viper.BindEnv("port", "PORT")
	rootCmd.PersistentFlags().StringSliceP("trusted_proxies", "t", []string{}, "trusted proxies IP addresses (separated by commas)")
	viper.BindPFlag("trusted_proxies", rootCmd.PersistentFlags().Lookup("trusted_proxies"))

	rootCmd.PersistentFlags().String("log_format", "text", "logger format")
	viper.BindPFlag("log_format", rootCmd.PersistentFlags().Lookup("log_format"))
	rootCmd.PersistentFlags().StringP("log_level", "l", "info", "logger level")
	viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log_level"))

	rootCmd.Flags().String("basic_auth_username", "", "basic auth username (required if basic auth password is set)")
	viper.BindPFlag("basic_auth_username", rootCmd.PersistentFlags().Lookup("basic_auth_username"))
	rootCmd.Flags().String("basic_auth_password", "", "basic auth password (required if basic auth username is set)")
	viper.BindPFlag("basic_auth_password", rootCmd.PersistentFlags().Lookup("basic_auth_password"))
	rootCmd.MarkFlagsRequiredTogether("basic_auth_username", "basic_auth_password")

	rootCmd.Flags().String("bearer_auth_token", "", "bearer authentication token")
	viper.BindPFlag("bearer_auth_token", rootCmd.PersistentFlags().Lookup("bearer_auth_token"))
}
