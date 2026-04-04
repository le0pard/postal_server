package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	gopostalExpand "github.com/openvenues/gopostal/expand"
	gopostalParser "github.com/openvenues/gopostal/parser"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

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
	if langs, ok := queryParams["languages"]; ok {
		options.Languages = langs
	}
	if val, ok := queryParams["latin_ascii"]; ok && len(val) > 0 {
		options.LatinAscii = stringToBool(val[0])
	}
	if val, ok := queryParams["transliterate"]; ok && len(val) > 0 {
		options.Transliterate = stringToBool(val[0])
	}
	if val, ok := queryParams["strip_accents"]; ok && len(val) > 0 {
		options.StripAccents = stringToBool(val[0])
	}
	if val, ok := queryParams["lowercase"]; ok && len(val) > 0 {
		options.Lowercase = stringToBool(val[0])
	}
	if val, ok := queryParams["trim_string"]; ok && len(val) > 0 {
		options.TrimString = stringToBool(val[0])
	}
	if val, ok := queryParams["replace_word_hyphens"]; ok && len(val) > 0 {
		options.ReplaceWordHyphens = stringToBool(val[0])
	}
	if val, ok := queryParams["delete_word_hyphens"]; ok && len(val) > 0 {
		options.DeleteWordHyphens = stringToBool(val[0])
	}
	if val, ok := queryParams["replace_numeric_hyphens"]; ok && len(val) > 0 {
		options.ReplaceNumericHyphens = stringToBool(val[0])
	}
	if val, ok := queryParams["delete_numeric_hyphens"]; ok && len(val) > 0 {
		options.DeleteNumericHyphens = stringToBool(val[0])
	}
	if val, ok := queryParams["split_alpha_from_numeric"]; ok && len(val) > 0 {
		options.SplitAlphaFromNumeric = stringToBool(val[0])
	}
	if val, ok := queryParams["delete_final_periods"]; ok && len(val) > 0 {
		options.DeleteFinalPeriods = stringToBool(val[0])
	}
	if val, ok := queryParams["delete_acronym_periods"]; ok && len(val) > 0 {
		options.DeleteAcronymPeriods = stringToBool(val[0])
	}
	if val, ok := queryParams["drop_english_possessives"]; ok && len(val) > 0 {
		options.DropEnglishPossessives = stringToBool(val[0])
	}
	if val, ok := queryParams["delete_apostrophes"]; ok && len(val) > 0 {
		options.DeleteApostrophes = stringToBool(val[0])
	}
	if val, ok := queryParams["expand_numex"]; ok && len(val) > 0 {
		options.ExpandNumex = stringToBool(val[0])
	}
	if val, ok := queryParams["roman_numerals"]; ok && len(val) > 0 {
		options.RomanNumerals = stringToBool(val[0])
	}

	if newComponents, found := parseAddressComponents(queryParams); found {
		options.AddressComponents = newComponents
	}

	return options
}

func parseAddressComponents(queryParams url.Values) (uint16, bool) {
	var components uint16 = gopostalExpand.AddressNone
	var found bool = false

	// Iterate over the valid keys we support, NOT the keys the user sent
	for key, component := range queryParamToAddressComponent {
		if values, ok := queryParams[key]; ok && len(values) > 0 {
			// Ensure we check if the value actually evaluates to true
			if stringToBool(values[0]) {
				found = true
				components |= component
			}
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

func SetupRouter() *gin.Engine {
	r := gin.New()

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

	return r
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
		r := SetupRouter()

		var handler http.Handler = r

		// If H2C is enabled in the config, wrap the router with the H2C handler
		if viper.GetBool("h2c") {
			log.Info().Msg("H2C (HTTP/2 Cleartext) enabled")
			h2s := &http2.Server{
				// How long the HTTP/2 connection can be completely idle before closing
				IdleTimeout: 120 * time.Second,
				// If there is no read activity, send a PING frame to the client
				// to check if they are still alive
				ReadIdleTimeout: 30 * time.Second,
			}

			handler = h2c.NewHandler(r, h2s)
		}

		srv := &http.Server{
			Addr:         fmt.Sprintf("%s:%d", viper.GetString("host"), viper.GetInt("port")),
			Handler:      handler,
			ReadTimeout:  30 * time.Second,  // Max time to read request headers/body
			WriteTimeout: 30 * time.Second,  // Max time to process and send the response
			IdleTimeout:  120 * time.Second, // Max time to keep a Keep-Alive connection open
		}

		go func() {
			log.Info().Msgf("Starting server on %s", srv.Addr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatal().Err(err).Msg("listen failed")
			}
		}()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		// Block until we receive our signal
		<-quit

		log.Info().Msg("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal().Err(err).Msg("Server forced to shutdown")
		}

		log.Info().Msg("Server exiting")
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
